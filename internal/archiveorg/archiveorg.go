// Package archiveorg searches archive.org for comic book archives and
// downloads them.
package archiveorg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	DefaultBaseURL = "https://archive.org"
	// Items per search page. Each item's file list is fetched separately.
	PerPage = 20

	metadataTTL     = 10 * time.Minute
	maxCacheEntries = 1000
	// Search results cap out at 10,000 items.
	maxPage = 10000 / PerPage
)

var (
	ErrNotFound = errors.New("file not found on archive.org")
	ErrTooLarge = errors.New("file is too large")
)

var identifierRegexp = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// File is a downloadable CBR/CBZ file in an archive.org item.
type File struct {
	Identifier string `json:"identifier"`
	ItemTitle  string `json:"itemTitle"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
}

type Results struct {
	Files      []File `json:"files"`
	TotalItems int    `json:"totalItems"`
	Page       int    `json:"page"`
	PerPage    int    `json:"perPage"`
}

type Client struct {
	BaseURL   string
	userAgent string
	api       *http.Client
	download  *http.Client

	mu    sync.Mutex
	cache map[string]cachedItem
}

type cachedItem struct {
	files   []File
	expires time.Time
}

func New(baseURL, userAgent string) *Client {
	c := &Client{
		BaseURL:   strings.TrimSuffix(baseURL, "/"),
		userAgent: userAgent,
		api:       &http.Client{Timeout: 30 * time.Second},
		cache:     map[string]cachedItem{},
	}
	// Downloads can be large, so they're bounded by the request context
	// rather than a fixed timeout.
	c.download = &http.Client{
		Transport:     &http.Transport{Proxy: http.ProxyFromEnvironment, ResponseHeaderTimeout: 30 * time.Second},
		CheckRedirect: c.checkRedirect,
	}
	return c
}

// checkRedirect only follows redirects to the archive.org host or its
// subdomains (where files are served from), over the same scheme.
func (c *Client) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return errors.New("too many redirects")
	}
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	host := req.URL.Hostname()
	if req.URL.Scheme != base.Scheme || (host != base.Hostname() && !strings.HasSuffix(host, "."+base.Hostname())) {
		return fmt.Errorf("refusing redirect to %s", req.URL.Redacted())
	}
	return nil
}

// Sort orders for Search, mapped to archive.org sort fields. Relevance (the
// default) sends no sort.
var sorts = map[string]string{
	"downloads": "downloads desc",
	"newest":    "addeddate desc",
	"title":     "titleSorter asc",
}

// fieldSyntax matches a Lucene field query like creator:"Frank Miller" or
// title:batman (but not "Batman: Year One").
var fieldSyntax = regexp.MustCompile(`(^|[\s(])[A-Za-z_][\w.-]*:[^\s]`)

// BuildQuery turns a user's search into an archive.org query for comic
// archives. Plain text is matched against titles, subjects and creators
// (weighted toward titles) rather than the full text of items, which
// includes OCR text and reviews. Queries using field syntax are passed
// through as written.
func BuildQuery(query string) string {
	const formats = `format:("Comic Book ZIP" OR "Comic Book RAR")`
	if fieldSyntax.MatchString(query) {
		return "(" + query + ") AND " + formats
	}
	q := escapeQuery(query)
	return "(title:(" + q + ")^5 OR subject:(" + q + ")^2 OR creator:(" + q + ")) AND " + formats
}

// escapeQuery escapes Lucene syntax in plain text, keeping balanced double
// quotes so phrases still work.
func escapeQuery(s string) string {
	keepQuotes := strings.Count(s, `"`)%2 == 0
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '"' && keepQuotes:
		case strings.ContainsRune(`+-&|!(){}[]^"~*?:\/`, r):
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Search finds items containing CBR/CBZ files and lists those files. sort is
// "downloads", "newest", "title", or anything else for relevance.
func (c *Client) Search(ctx context.Context, query string, page int, sort string) (*Results, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("search query is empty")
	}
	page = min(max(page, 1), maxPage)

	q := url.Values{
		"q":      {BuildQuery(query)},
		"fl[]":   {"identifier", "title"},
		"rows":   {strconv.Itoa(PerPage)},
		"page":   {strconv.Itoa(page)},
		"output": {"json"},
	}
	if s, ok := sorts[sort]; ok {
		q.Set("sort[]", s)
	}
	var resp struct {
		Response struct {
			NumFound int `json:"numFound"`
			Docs     []struct {
				Identifier string `json:"identifier"`
				Title      text   `json:"title"`
			} `json:"docs"`
		} `json:"response"`
	}
	if err := c.getJSON(ctx, "/advancedsearch.php?"+q.Encode(), &resp); err != nil {
		return nil, err
	}

	// Fetch each item's file list, keeping results in search order.
	docs := resp.Response.Docs
	itemFiles := make([][]File, len(docs))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(6)
	for i, doc := range docs {
		g.Go(func() error {
			files, err := c.itemFiles(gctx, doc.Identifier)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				// One unavailable item shouldn't fail the whole search.
				slog.Warn("archive.org item metadata", "identifier", doc.Identifier, "err", err)
				return nil
			}
			itemFiles[i] = files
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	results := &Results{Files: []File{}, TotalItems: resp.Response.NumFound, Page: page, PerPage: PerPage}
	for _, files := range itemFiles {
		results.Files = append(results.Files, files...)
	}
	return results, nil
}

// ItemFile returns a downloadable CBR/CBZ file in an item.
func (c *Client) ItemFile(ctx context.Context, identifier, name string) (*File, error) {
	if !identifierRegexp.MatchString(identifier) {
		return nil, ErrNotFound
	}
	files, err := c.itemFiles(ctx, identifier)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if f.Name == name {
			return &f, nil
		}
	}
	return nil, ErrNotFound
}

// Download writes a file to w, failing with ErrTooLarge past maxBytes.
func (c *Client) Download(ctx context.Context, f *File, w io.Writer, maxBytes int64) error {
	segments := strings.Split(f.Name, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	u := c.BaseURL + "/download/" + url.PathEscape(f.Identifier) + "/" + strings.Join(segments, "/")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.userAgent)
	res, err := c.download.Do(req)
	if err != nil {
		return fmt.Errorf("downloading from archive.org: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("archive.org returned %s", res.Status)
	}
	if res.ContentLength > maxBytes {
		return ErrTooLarge
	}
	n, err := io.Copy(w, io.LimitReader(res.Body, maxBytes+1))
	if err != nil {
		return fmt.Errorf("downloading from archive.org: %w", err)
	}
	if n > maxBytes {
		return ErrTooLarge
	}
	return nil
}

// itemFiles returns an item's downloadable CBR/CBZ files, cached briefly.
func (c *Client) itemFiles(ctx context.Context, identifier string) ([]File, error) {
	c.mu.Lock()
	cached, ok := c.cache[identifier]
	c.mu.Unlock()
	if ok && time.Now().Before(cached.expires) {
		return cached.files, nil
	}

	var item struct {
		Metadata struct {
			Title text `json:"title"`
		} `json:"metadata"`
		Files []struct {
			Name    string `json:"name"`
			Format  string `json:"format"`
			Size    string `json:"size"`
			Private string `json:"private"`
		} `json:"files"`
	}
	if err := c.getJSON(ctx, "/metadata/"+url.PathEscape(identifier), &item); err != nil {
		return nil, err
	}

	files := []File{}
	for _, f := range item.Files {
		// Private files belong to restricted items and can't be downloaded.
		if f.Private == "true" || !isComic(f.Name, f.Format) {
			continue
		}
		size, _ := strconv.ParseInt(f.Size, 10, 64)
		title := string(item.Metadata.Title)
		if title == "" {
			title = identifier
		}
		files = append(files, File{Identifier: identifier, ItemTitle: title, Name: f.Name, Size: size})
	}

	c.mu.Lock()
	if len(c.cache) >= maxCacheEntries {
		c.cache = map[string]cachedItem{}
	}
	c.cache[identifier] = cachedItem{files: files, expires: time.Now().Add(metadataTTL)}
	c.mu.Unlock()
	return files, nil
}

func isComic(name, format string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".cbz", ".cbr":
		return true
	}
	return format == "Comic Book ZIP" || format == "Comic Book RAR"
}

func (c *Client) getJSON(ctx context.Context, pathAndQuery string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+pathAndQuery, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.userAgent)
	res, err := c.api.Do(req)
	if err != nil {
		return fmt.Errorf("contacting archive.org: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("archive.org returned %s", res.Status)
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 64<<20)).Decode(v); err != nil {
		return fmt.Errorf("reading archive.org response: %w", err)
	}
	return nil
}

// text is a metadata field, which archive.org returns as either a string or
// a list of strings.
type text string

func (t *text) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*t = text(s)
		return nil
	}
	var list []string
	if err := json.Unmarshal(b, &list); err != nil {
		return nil // Ignore unexpected shapes rather than failing the item.
	}
	*t = text(strings.Join(list, ", "))
	return nil
}

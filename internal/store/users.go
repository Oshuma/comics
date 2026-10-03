package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/netip"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

// Same cost the Rails app used with Devise, so hashes are interchangeable.
const bcryptCost = 11

var emailRegexp = regexp.MustCompile(`^[^@\s]+@[^@\s]+$`)

var dummyHash = sync.OnceValue(func() []byte {
	h, _ := bcrypt.GenerateFromPassword([]byte("dummy password"), bcryptCost)
	return h
})

type User struct {
	ID          int       `json:"id"`
	Email       string    `json:"email"`
	Admin       bool      `json:"admin"`
	CreatedAt   time.Time `json:"createdAt"`
	GroupCount  int       `json:"groupCount"`
	ComicCount  int       `json:"comicCount"`
	passwordHash string
}

type NewUser struct {
	Email                string `json:"email"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"passwordConfirmation"`
	Admin                bool   `json:"admin"`
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) CreateUser(ctx context.Context, u NewUser) (*User, error) {
	email := normalizeEmail(u.Email)

	var msgs []string
	switch {
	case email == "":
		msgs = append(msgs, "Email can't be blank")
	case !emailRegexp.MatchString(email):
		msgs = append(msgs, "Email is invalid")
	}
	switch {
	case u.Password == "":
		msgs = append(msgs, "Password can't be blank")
	case len(u.Password) < 8:
		msgs = append(msgs, "Password is too short (minimum is 8 characters)")
	case len(u.Password) > 72:
		msgs = append(msgs, "Password is too long (maximum is 72 characters)")
	}
	if u.Password != u.PasswordConfirmation {
		msgs = append(msgs, "Password confirmation doesn't match Password")
	}
	if len(msgs) > 0 {
		return nil, invalid(msgs...)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcryptCost)
	if err != nil {
		return nil, err
	}

	user := &User{Email: email, Admin: u.Admin}
	err = s.db.QueryRow(ctx, `
		INSERT INTO users (email, encrypted_password, admin, created_at, updated_at)
		VALUES ($1, $2, $3, `+now+`, `+now+`)
		RETURNING id, created_at`,
		email, string(hash), u.Admin,
	).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, invalid("Email has already been taken")
		}
		return nil, err
	}
	return user, nil
}

func (s *Store) GetUser(ctx context.Context, id int) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(ctx,
		`SELECT id, email, COALESCE(admin, false), created_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Admin, &u.CreatedAt)
	if err != nil {
		return nil, notFound(err)
	}
	return u, nil
}

// Authenticate checks the credentials and records the sign-in.
func (s *Store) Authenticate(ctx context.Context, email, password string, ip netip.Addr) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(ctx,
		`SELECT id, email, COALESCE(admin, false), created_at, encrypted_password FROM users WHERE email = $1`,
		normalizeEmail(email),
	).Scan(&u.ID, &u.Email, &u.Admin, &u.CreatedAt, &u.passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		// Burn comparable time so response timing doesn't reveal valid emails.
		bcrypt.CompareHashAndPassword(dummyHash(), []byte(password))
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.passwordHash), []byte(password)) != nil {
		return nil, ErrNotFound
	}

	var addr *netip.Addr
	if ip.IsValid() {
		addr = &ip
	}
	_, err = s.db.Exec(ctx, `
		UPDATE users SET
			sign_in_count = sign_in_count + 1,
			last_sign_in_at = COALESCE(current_sign_in_at, `+now+`),
			last_sign_in_ip = COALESCE(current_sign_in_ip, $2),
			current_sign_in_at = `+now+`,
			current_sign_in_ip = $2
		WHERE id = $1`, u.ID, addr)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Store) ListUsers(ctx context.Context, page int) ([]User, int, error) {
	var total int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT u.id, u.email, COALESCE(u.admin, false), u.created_at,
			(SELECT count(*) FROM groups g WHERE g.user_id = u.id),
			(SELECT count(*) FROM comics c WHERE c.user_id = u.id)
		FROM users u
		ORDER BY u.created_at ASC, u.id ASC
		LIMIT $1 OFFSET $2`, PerPage, offset(page))
	if err != nil {
		return nil, 0, err
	}
	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (User, error) {
		var u User
		err := row.Scan(&u.ID, &u.Email, &u.Admin, &u.CreatedAt, &u.GroupCount, &u.ComicCount)
		return u, err
	})
	return users, total, err
}

func (s *Store) SetAdmin(ctx context.Context, id int, admin bool) error {
	tag, err := s.db.Exec(ctx, `UPDATE users SET admin = $2, updated_at = `+now+` WHERE id = $1`, id, admin)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteUser removes a user along with all of their groups, comics and history.
func (s *Store) DeleteUser(ctx context.Context, id int) error {
	return s.deleteTx(ctx, func(tx pgx.Tx) ([]int, error) {
		comicIDs, err := collectIDs(ctx, tx, `SELECT id FROM comics WHERE user_id = $1`, id)
		if err != nil {
			return nil, err
		}
		pageIDs, err := s.deleteComics(ctx, tx, comicIDs)
		if err != nil {
			return nil, err
		}
		for _, sql := range []string{
			`DELETE FROM histories WHERE user_id = $1`,
			`DELETE FROM groups WHERE user_id = $1`,
			`DELETE FROM sessions WHERE user_id = $1`,
		} {
			if _, err := tx.Exec(ctx, sql, id); err != nil {
				return nil, err
			}
		}
		tag, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			return nil, ErrNotFound
		}
		return pageIDs, nil
	})
}

// Sessions

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// CreateSession returns a new random session token for the user.
func (s *Store) CreateSession(ctx context.Context, userID int, ttl time.Duration) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)

	_, err := s.db.Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at, created_at)
		VALUES ($1, $2, `+now+` + make_interval(secs => $3), `+now+`)`,
		hashToken(token), userID, ttl.Seconds())
	if err != nil {
		return "", err
	}

	// Opportunistic cleanup of expired sessions.
	s.db.Exec(ctx, `DELETE FROM sessions WHERE expires_at < `+now)

	return token, nil
}

func (s *Store) SessionUser(ctx context.Context, token string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(ctx, `
		SELECT u.id, u.email, COALESCE(u.admin, false), u.created_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > `+now,
		hashToken(token),
	).Scan(&u.ID, &u.Email, &u.Admin, &u.CreatedAt)
	if err != nil {
		return nil, notFound(err)
	}
	return u, nil
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hashToken(token))
	return err
}

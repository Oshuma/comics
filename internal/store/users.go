package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 11

var emailRegexp = regexp.MustCompile(`^[^@\s]+@[^@\s]+$`)

var dummyHash = sync.OnceValue(func() []byte {
	h, _ := bcrypt.GenerateFromPassword([]byte("dummy password"), bcryptCost)
	return h
})

type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	Admin        bool      `json:"admin"`
	CreatedAt    time.Time `json:"createdAt"`
	GroupCount   int       `json:"groupCount"`
	ComicCount   int       `json:"comicCount"`
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
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM users`).Scan(&n)
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
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash, admin) VALUES (?, ?, ?) RETURNING id, created_at`,
		email, string(hash), u.Admin,
	).Scan(&user.ID, &user.CreatedAt)
	if isUniqueViolation(err) {
		return nil, invalid("Email has already been taken")
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Store) GetUser(ctx context.Context, id int) (*User, error) {
	u := &User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, admin, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.Admin, &u.CreatedAt)
	if err != nil {
		return nil, notFound(err)
	}
	return u, nil
}

// Authenticate checks the credentials.
func (s *Store) Authenticate(ctx context.Context, email, password string) (*User, error) {
	u := &User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, admin, created_at, password_hash FROM users WHERE email = ?`,
		normalizeEmail(email),
	).Scan(&u.ID, &u.Email, &u.Admin, &u.CreatedAt, &u.passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
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
	return u, nil
}

func (s *Store) ListUsers(ctx context.Context, page int) ([]User, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.email, u.admin, u.created_at,
			(SELECT count(*) FROM groups g WHERE g.user_id = u.id),
			(SELECT count(*) FROM comics c WHERE c.user_id = u.id)
		FROM users u
		ORDER BY u.id
		LIMIT ? OFFSET ?`, PerPage, offset(page))
	users, err := collect(rows, err, func(row scanner) (User, error) {
		var u User
		err := row.Scan(&u.ID, &u.Email, &u.Admin, &u.CreatedAt, &u.GroupCount, &u.ComicCount)
		return u, err
	})
	return users, total, err
}

func (s *Store) SetAdmin(ctx context.Context, id int, admin bool) error {
	return requireAffected(s.db.ExecContext(ctx, `UPDATE users SET admin = ? WHERE id = ?`, admin, id))
}

// DeleteUser removes a user along with all of their groups, comics and history.
func (s *Store) DeleteUser(ctx context.Context, id int) error {
	return s.deleteComics(ctx, func(tx *sql.Tx) ([]int, error) {
		comicIDs, err := deletedIDs(ctx, tx, `DELETE FROM comics WHERE user_id = ? RETURNING id`, id)
		if err != nil {
			return nil, err
		}
		err = requireAffected(tx.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id))
		return comicIDs, err
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

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, datetime('now', ?))`,
		hashToken(token), userID, fmt.Sprintf("%+d seconds", int(ttl.Seconds())))
	if err != nil {
		return "", err
	}

	// Opportunistic cleanup of expired sessions.
	s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < datetime('now')`)

	return token, nil
}

func (s *Store) SessionUser(ctx context.Context, token string) (*User, error) {
	u := &User{}
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.admin, u.created_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > datetime('now')`,
		hashToken(token),
	).Scan(&u.ID, &u.Email, &u.Admin, &u.CreatedAt)
	if err != nil {
		return nil, notFound(err)
	}
	return u, nil
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, hashToken(token))
	return err
}

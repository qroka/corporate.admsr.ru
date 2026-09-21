package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	SessionCookie = "corp_session"
	DefaultTTLHours = 24
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

type User struct {
	ID        int64
	Login     string
	Firstname string
	Surname   string
	Lastname  string
	Email     *string
	Phone     *string
	OFO       any
	UserGroup string
	Role      string
	Status    bool
	Auth      bool
}

type Service struct {
	Pool     *pgxpool.Pool
	TTLHours int
}

func (s *Service) ttl() time.Duration {
	h := s.TTLHours
	if h <= 0 {
		h = DefaultTTLHours
	}
	return time.Duration(h) * time.Hour
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func ExtractToken(r *http.Request) string {
	if hdr := r.Header.Get("Authorization"); hdr != "" {
		const prefix = "Bearer "
		if len(hdr) > len(prefix) && strings.EqualFold(hdr[:len(prefix)], prefix) {
			return strings.TrimSpace(hdr[len(prefix):])
		}
	}
	if x := strings.TrimSpace(r.Header.Get("X-Session-Token")); x != "" {
		return x
	}
	if c, err := r.Cookie(SessionCookie); err == nil && c.Value != "" {
		return c.Value
	}
	return ""
}

func (s *Service) CreateSession(ctx context.Context, w http.ResponseWriter, r *http.Request, userID int64) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)
	hash := HashToken(token)

	var ip *string
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil && host != "" {
		ip = &host
	} else if r.RemoteAddr != "" {
		ip = &r.RemoteAddr
	}

	var ua *string
	if u := r.UserAgent(); u != "" {
		if utf8.RuneCountInString(u) > 500 {
			runes := []rune(u)
			u = string(runes[:500])
		}
		ua = &u
	}

	ttlHours := s.TTLHours
	if ttlHours <= 0 {
		ttlHours = DefaultTTLHours
	}

	_, err := s.Pool.Exec(ctx, `
		INSERT INTO public.user_sessions (user_id, token_hash, expires_at, ip_address, user_agent)
		VALUES ($1, $2, now() + make_interval(hours => $3), $4, $5)`,
		userID, hash, ttlHours, ip, ua,
	)
	if err != nil {
		return "", err
	}

	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(s.ttl()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	return token, nil
}

func (s *Service) RevokeSession(ctx context.Context, w http.ResponseWriter, token string) {
	if token != "" {
		_, _ = s.Pool.Exec(ctx, `
			UPDATE public.user_sessions SET revoked_at = now()
			WHERE token_hash = $1 AND revoked_at IS NULL`, HashToken(token))
	}
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Service) SessionUserID(ctx context.Context, token string) (int64, bool) {
	if token == "" {
		return 0, false
	}
	var uid int64
	err := s.Pool.QueryRow(ctx, `
		SELECT user_id FROM public.user_sessions
		WHERE token_hash = $1 AND revoked_at IS NULL LIMIT 1`, HashToken(token)).Scan(&uid)
	if err != nil {
		return 0, false
	}
	return uid, true
}

func (s *Service) CurrentUser(ctx context.Context, r *http.Request) (*User, error) {
	token := ExtractToken(r)
	if token == "" {
		return nil, ErrUnauthorized
	}

	var (
		sessionID int64
		expiresAt time.Time
		u         User
	)
	err := s.Pool.QueryRow(ctx, `
		SELECT s.id, s.expires_at, u.id, u.login, u.firstname, u.surname, u.lastname,
		       u.email, u.phone, u.ofo, u.user_group, u.role, u.status, u.auth
		FROM public.user_sessions s
		JOIN public.user_info u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.revoked_at IS NULL
		LIMIT 1`, HashToken(token)).Scan(
		&sessionID, &expiresAt,
		&u.ID, &u.Login, &u.Firstname, &u.Surname, &u.Lastname,
		&u.Email, &u.Phone, &u.OFO, &u.UserGroup, &u.Role, &u.Status, &u.Auth,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}

	if !u.Status || !u.Auth || !expiresAt.After(time.Now()) {
		_, _ = s.Pool.Exec(ctx, `UPDATE public.user_sessions SET revoked_at = now() WHERE id = $1`, sessionID)
		return nil, ErrUnauthorized
	}

	_, _ = s.Pool.Exec(ctx, `UPDATE public.user_sessions SET last_seen_at = now() WHERE id = $1`, sessionID)
	_, _ = s.Pool.Exec(ctx, `UPDATE public.user_info SET last_activity = now() WHERE id = $1`, u.ID)

	u.Status = true
	u.Auth = true
	return &u, nil
}

func (s *Service) RequireUser(ctx context.Context, r *http.Request) (*User, error) {
	return s.CurrentUser(ctx, r)
}

func (s *Service) RequireAdmin(ctx context.Context, r *http.Request) (*User, error) {
	u, err := s.RequireUser(ctx, r)
	if err != nil {
		return nil, err
	}
	if u.UserGroup != "admin" {
		return nil, ErrForbidden
	}
	return u, nil
}

func IsAdmin(u *User) bool {
	return u != nil && u.UserGroup == "admin"
}

func (s *Service) RequireSection(ctx context.Context, r *http.Request, section string) (*User, error) {
	u, err := s.RequireUser(ctx, r)
	if err != nil {
		return nil, err
	}
	if !CanEditSection(ctx, s.Pool, u, section) {
		return nil, ErrForbidden
	}
	return u, nil
}

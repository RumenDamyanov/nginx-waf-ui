package session

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"
)

const cookieName = "waf_session"

// Store manages sessions in memory.
type Store struct {
	secret []byte
	maxAge int
	mu     sync.RWMutex
	data   map[string]*Session
}

// Session holds per-user session data.
type Session struct {
	ID       string
	Username string
	Created  time.Time
}

// NewStore creates a session store.
func NewStore(secret string, maxAge int) *Store {
	return &Store{
		secret: []byte(secret),
		maxAge: maxAge,
		data:   make(map[string]*Session),
	}
}

// Create creates a new session for the given username and sets the cookie.
func (s *Store) Create(w http.ResponseWriter, username string) {
	id := generateID()
	sig := s.sign(id)
	token := id + "." + sig

	s.mu.Lock()
	s.data[id] = &Session{
		ID:       id,
		Username: username,
		Created:  time.Now(),
	}
	s.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   s.maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   false,
	})
}

// Get returns the current session from the request, or nil.
func (s *Store) Get(r *http.Request) *Session {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return nil
	}

	parts := strings.SplitN(cookie.Value, ".", 2)
	if len(parts) != 2 {
		return nil
	}
	id, sig := parts[0], parts[1]

	if !hmac.Equal([]byte(s.sign(id)), []byte(sig)) {
		return nil
	}

	s.mu.RLock()
	sess := s.data[id]
	s.mu.RUnlock()

	if sess == nil {
		return nil
	}

	if time.Since(sess.Created) > time.Duration(s.maxAge)*time.Second {
		s.mu.Lock()
		delete(s.data, id)
		s.mu.Unlock()
		return nil
	}

	return sess
}

// Destroy removes the session.
func (s *Store) Destroy(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return
	}
	parts := strings.SplitN(cookie.Value, ".", 2)
	if len(parts) == 2 {
		s.mu.Lock()
		delete(s.data, parts[0])
		s.mu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

func (s *Store) sign(id string) string {
	h := hmac.New(sha256.New, s.secret)
	h.Write([]byte(id))
	return hex.EncodeToString(h.Sum(nil))
}

func generateID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

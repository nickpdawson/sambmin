package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"time"
)

// Session represents an authenticated user session.
type Session struct {
	ID        string
	Username  string
	DN        string
	Groups    []string
	CSRFToken string
	// encPW holds the user's password encrypted with the server key.
	// Used for write operations via samba-tool.
	encPW   []byte
	Expires time.Time
}

// Store manages user sessions in memory.
// In-memory; sessions are lost on restart.
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	gcm      cipher.AEAD
	timeout  time.Duration
}

// NewStore creates a session store with AES-GCM encryption for passwords.
func NewStore(timeoutHours int) (*Store, error) {
	// Generate random 256-bit key for AES-GCM encryption of stored passwords
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate session key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	timeout := time.Duration(timeoutHours) * time.Hour
	if timeout <= 0 {
		timeout = 8 * time.Hour
	}

	s := &Store{
		sessions: make(map[string]*Session),
		gcm:      gcm,
		timeout:  timeout,
	}

	// Background cleanup of expired sessions every 5 minutes
	go s.cleanupLoop()

	return s, nil
}

// Create creates a new session for an authenticated user.
// The password is encrypted before storage.
func (s *Store) Create(username, dn string, groups []string, password string) (*Session, error) {
	id, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	csrfToken, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("generate CSRF token: %w", err)
	}

	encPW, err := s.encryptPassword(password)
	if err != nil {
		return nil, fmt.Errorf("encrypt session password: %w", err)
	}

	sess := &Session{
		ID:        id,
		Username:  username,
		DN:        dn,
		Groups:    groups,
		CSRFToken: csrfToken,
		encPW:     encPW,
		Expires:   time.Now().Add(s.timeout),
	}

	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()

	return sess, nil
}

// SetTimeout updates the session timeout duration. New sessions will use the
// new timeout; existing sessions keep their original expiry.
func (s *Store) SetTimeout(hours int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := time.Duration(hours) * time.Hour
	if t <= 0 {
		t = 8 * time.Hour
	}
	s.timeout = t
}

// Get retrieves a session by ID. Returns nil if not found or expired.
func (s *Store) Get(id string) *Session {
	s.mu.RLock()
	sess, ok := s.sessions[id]
	s.mu.RUnlock()

	if !ok {
		return nil
	}
	if time.Now().After(sess.Expires) {
		s.Delete(id)
		return nil
	}
	return sess
}

// Delete removes a session.
func (s *Store) Delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// Password decrypts and returns the stored password for write operations.
func (s *Store) Password(sess *Session) (string, error) {
	if sess == nil || len(sess.encPW) == 0 {
		return "", fmt.Errorf("no session credentials")
	}
	return s.decryptPassword(sess.encPW)
}

func (s *Store) encryptPassword(pw string) ([]byte, error) {
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return s.gcm.Seal(nonce, nonce, []byte(pw), nil), nil
}

func (s *Store) decryptPassword(data []byte) (string, error) {
	nonceSize := s.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := s.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}

func (s *Store) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, sess := range s.sessions {
			if now.After(sess.Expires) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", fmt.Errorf("generate session ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

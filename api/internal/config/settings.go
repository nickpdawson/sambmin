package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// SettingsOverlay holds GUI-managed overrides that layer on top of config.yaml.
// Only non-nil fields override the base config.
type SettingsOverlay struct {
	Connection  *ConnectionOverlay  `json:"connection,omitempty"`
	Auth        *AuthOverlay        `json:"auth,omitempty"`
	RBAC        *RBACOverlay        `json:"rbac,omitempty"`
	Application *ApplicationOverlay `json:"application,omitempty"`
}

type ConnectionOverlay struct {
	DCs      []DCConfig `json:"domain_controllers,omitempty"`
	BaseDN   string     `json:"base_dn,omitempty"`
	Protocol string     `json:"protocol,omitempty"`
}

type AuthOverlay struct {
	Kerberos       *KerberosOverlay `json:"kerberos,omitempty"`
	SessionTimeout *int             `json:"session_timeout,omitempty"` // pointer: distinguish 0 from absent
}

type KerberosOverlay struct {
	Enabled    *bool  `json:"enabled,omitempty"`
	KeytabPath string `json:"keytab_path,omitempty"`
}

type RBACOverlay struct {
	Roles []RoleMapping `json:"roles"`
}

type RoleMapping struct {
	Role        string   `json:"role"`
	Groups      []string `json:"groups"`
	Permissions []string `json:"permissions"`
}

type ApplicationOverlay struct {
	AuditRetentionDays *int `json:"audit_retention_days,omitempty"`
}

// SettingsStore manages loading, saving, and applying the settings overlay.
type SettingsStore struct {
	mu      sync.RWMutex
	path    string
	overlay SettingsOverlay
}

// SettingsPath derives the settings.json path from a config.yaml path.
func SettingsPath(configPath string) string {
	dir := filepath.Dir(configPath)
	return filepath.Join(dir, "settings.json")
}

// NewSettingsStore loads an existing settings.json or starts with an empty overlay.
func NewSettingsStore(path string) (*SettingsStore, error) {
	s := &SettingsStore{path: path}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil // start empty
		}
		return nil, fmt.Errorf("read settings: %w", err)
	}

	if err := json.Unmarshal(data, &s.overlay); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}

	return s, nil
}

// Save atomically writes the overlay to disk (temp file + rename, 0640 permissions).
func (s *SettingsStore) Save() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.overlay, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, "settings-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Chmod(tmpName, 0640); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := os.Rename(tmpName, s.path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename settings: %w", err)
	}

	return nil
}

// Overlay returns a copy of the current overlay (thread-safe).
func (s *SettingsStore) Overlay() SettingsOverlay {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.overlay
}

// ApplyTo returns a new Config with overlay values merged on top of base.
// The base config is not mutated.
func (s *SettingsStore) ApplyTo(base *Config) *Config {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Shallow copy
	merged := *base

	if c := s.overlay.Connection; c != nil {
		if len(c.DCs) > 0 {
			merged.DCs = make([]DCConfig, len(c.DCs))
			copy(merged.DCs, c.DCs)
		}
		if c.BaseDN != "" {
			merged.BaseDN = c.BaseDN
		}
		// Protocol is informational for the UI; the actual LDAP connection
		// is determined by port numbers on each DC. We store it but don't
		// change connection behavior here.
	}

	if a := s.overlay.Auth; a != nil {
		if k := a.Kerberos; k != nil {
			if k.KeytabPath != "" {
				merged.Kerberos.KeytabPath = k.KeytabPath
			}
		}
		if a.SessionTimeout != nil {
			merged.SessionTimeout = *a.SessionTimeout
		}
	}

	// RBAC and Application overlays are consumed by the settings GET endpoint
	// rather than merged into the Config struct (they don't map to config.yaml fields).

	return &merged
}

// UpdateConnection replaces the connection overlay. Returns field names that require restart.
func (s *SettingsStore) UpdateConnection(c ConnectionOverlay) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var restartFields []string

	old := s.overlay.Connection
	if old == nil || !dcListEqual(old.DCs, c.DCs) {
		restartFields = append(restartFields, "domain_controllers")
	}
	if old == nil || old.BaseDN != c.BaseDN {
		restartFields = append(restartFields, "base_dn")
	}

	s.overlay.Connection = &c
	return restartFields
}

// UpdateAuth replaces the auth overlay. Returns field names that require restart.
func (s *SettingsStore) UpdateAuth(a AuthOverlay) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var restartFields []string

	if a.Kerberos != nil {
		old := s.overlay.Auth
		if old == nil || old.Kerberos == nil {
			restartFields = append(restartFields, "kerberos")
		} else if a.Kerberos.KeytabPath != old.Kerberos.KeytabPath {
			restartFields = append(restartFields, "kerberos")
		}
	}
	// Session timeout does NOT require restart — applied immediately.

	s.overlay.Auth = &a
	return restartFields
}

// UpdateRBAC replaces the RBAC overlay.
func (s *SettingsStore) UpdateRBAC(r RBACOverlay) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.overlay.RBAC = &r
}

// UpdateApplication replaces the application overlay.
func (s *SettingsStore) UpdateApplication(a ApplicationOverlay) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.overlay.Application = &a
}

// dcListEqual checks if two DC lists are equivalent.
func dcListEqual(a, b []DCConfig) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Hostname != b[i].Hostname ||
			a[i].Address != b[i].Address ||
			a[i].Port != b[i].Port ||
			a[i].Site != b[i].Site ||
			a[i].Primary != b[i].Primary {
			return false
		}
	}
	return true
}

// domainFromBaseDN converts "DC=example,DC=com" to "EXAMPLE.COM".
func domainFromBaseDNConfig(baseDN string) string {
	var parts []string
	for _, component := range strings.Split(baseDN, ",") {
		component = strings.TrimSpace(component)
		if strings.HasPrefix(strings.ToUpper(component), "DC=") {
			parts = append(parts, strings.ToUpper(component[3:]))
		}
	}
	return strings.Join(parts, ".")
}

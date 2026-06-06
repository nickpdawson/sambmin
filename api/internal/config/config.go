package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// Server
	BindAddr       string   `yaml:"bind_addr"`
	Port           int      `yaml:"port"`
	AllowedOrigins []string `yaml:"allowed_origins"`

	// Samba DC connection
	DCs []DCConfig `yaml:"domain_controllers"`

	// LDAP
	BaseDN string `yaml:"base_dn"`
	BindDN string `yaml:"bind_dn"` // Service account DN for LDAP queries
	BindPW string `yaml:"bind_pw"` // Can also use SAMBMIN_BIND_PW env var

	// Kerberos
	Kerberos KerberosConfig `yaml:"kerberos"`

	// Python scripts path
	ScriptsPath string `yaml:"scripts_path"`

	// Session
	SessionTimeout int `yaml:"session_timeout_hours"`

	// RFC2307 / POSIX attribute auto-assignment for new users and groups
	RFC2307 RFC2307Config `yaml:"rfc2307"`
}

// RFC2307Config controls auto-assignment of uidNumber/gidNumber and related
// POSIX attributes on new users and groups. Sambmin auto-detects whether the
// domain is already using RFC2307 by sampling existing users for uidNumber;
// these settings only kick in when detection (or an explicit override) says
// the domain is RFC2307-enabled.
type RFC2307Config struct {
	MinUID       int    `yaml:"min_uid"`       // Floor for allocated uidNumbers (default 10000)
	MinGID       int    `yaml:"min_gid"`       // Floor for allocated gidNumbers (default 10000)
	DefaultShell string `yaml:"default_shell"` // loginShell on new users (default /bin/sh)
	HomeTemplate string `yaml:"home_template"` // unixHomeDirectory; "%s" -> sAMAccountName (default /home/%s)
}

type DCConfig struct {
	Hostname string `yaml:"hostname"`
	Address  string `yaml:"address"`
	Site     string `yaml:"site"`
	Port     int    `yaml:"port"`
	Primary  bool   `yaml:"primary"`
}

type KerberosConfig struct {
	Realm      string `yaml:"realm"`
	KDC        string `yaml:"kdc"`
	KeytabPath string `yaml:"keytab_path"`
	// "heimdal" or "mit"
	Implementation string `yaml:"implementation"`
}

func Load() (*Config, string, error) {
	cfg := &Config{
		BindAddr:       "127.0.0.1",
		Port:           8443,
		AllowedOrigins: []string{"http://localhost:5173"},
		ScriptsPath:    "/usr/local/share/sambmin/scripts",
		SessionTimeout: 8,
		RFC2307: RFC2307Config{
			MinUID:       10000,
			MinGID:       10000,
			DefaultShell: "/bin/sh",
			HomeTemplate: "/home/%s",
		},
	}

	configPath := os.Getenv("SAMBMIN_CONFIG")
	if configPath == "" {
		configPath = "/usr/local/etc/sambmin/config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, configPath, nil // Use defaults if no config file
		}
		return nil, "", fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, "", fmt.Errorf("parse config: %w", err)
	}

	// Backfill RFC2307 defaults for any fields the user omitted, so a partial
	// rfc2307: block in config.yaml doesn't leave zero values that would
	// allocate uidNumber=1 or set loginShell="".
	if cfg.RFC2307.MinUID == 0 {
		cfg.RFC2307.MinUID = 10000
	}
	if cfg.RFC2307.MinGID == 0 {
		cfg.RFC2307.MinGID = 10000
	}
	if cfg.RFC2307.DefaultShell == "" {
		cfg.RFC2307.DefaultShell = "/bin/sh"
	}
	if cfg.RFC2307.HomeTemplate == "" {
		cfg.RFC2307.HomeTemplate = "/home/%s"
	}

	return cfg, configPath, nil
}

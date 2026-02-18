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

	// PostgreSQL
	Database DatabaseConfig `yaml:"database"`

	// Python scripts path
	ScriptsPath string `yaml:"scripts_path"`

	// Session
	SessionTimeout int `yaml:"session_timeout_hours"`
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

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"ssl_mode"`
}

func Load() (*Config, error) {
	cfg := &Config{
		BindAddr:       "127.0.0.1",
		Port:           8443,
		AllowedOrigins: []string{"http://localhost:5173"},
		ScriptsPath:    "/usr/local/share/sambmin/scripts",
		SessionTimeout: 8,
		Database: DatabaseConfig{
			Host:    "localhost",
			Port:    5432,
			Name:    "sambmin",
			SSLMode: "disable",
		},
	}

	configPath := os.Getenv("SAMBMIN_CONFIG")
	if configPath == "" {
		configPath = "/usr/local/etc/sambmin/config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Use defaults if no config file
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

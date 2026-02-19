package handlers

import (
	"net/http"
)

func handleGetSettingsMock(w http.ResponseWriter, _ *http.Request) {
	cfg := handlerConfig

	// Build DC list from real config
	dcs := make([]map[string]any, 0, len(cfg.DCs))
	for _, dc := range cfg.DCs {
		port := dc.Port
		if port == 0 {
			port = 636
		}
		dcs = append(dcs, map[string]any{
			"hostname": dc.Hostname,
			"address":  dc.Address,
			"port":     port,
			"site":     dc.Site,
			"primary":  dc.Primary,
			"status":   "configured",
		})
	}

	realm := cfg.Kerberos.Realm
	if realm == "" {
		realm = domainFromBaseDN(cfg.BaseDN)
	}

	sessionTimeout := cfg.SessionTimeout
	if sessionTimeout == 0 {
		sessionTimeout = 8
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"connection": map[string]any{
			"domainControllers": dcs,
			"baseDN":            cfg.BaseDN,
			"realm":             realm,
			"protocol":          "ldap",
		},
		"tls": map[string]any{
			"provider": "nginx-managed",
			"domain":   "sambmin.dzsec.net",
		},
		"auth": map[string]any{
			"kerberos": map[string]any{
				"enabled":        cfg.Kerberos.Implementation != "",
				"implementation": cfg.Kerberos.Implementation,
				"keytab":         cfg.Kerberos.KeytabPath,
			},
			"ldapBind": map[string]any{
				"enabled": cfg.BindDN != "",
			},
			"sessionTimeout": sessionTimeout,
		},
		"rbac": map[string]any{
			"roles": []map[string]any{
				{"role": "Full Admin", "groups": []string{"Domain Admins", "Enterprise Admins"}, "permissions": []string{"*"}},
				{"role": "User Admin", "groups": []string{"Account Operators"}, "permissions": []string{"users.*", "groups.read"}},
				{"role": "DNS Admin", "groups": []string{"DnsAdmins"}, "permissions": []string{"dns.*"}},
				{"role": "Read Only", "groups": []string{"Domain Users"}, "permissions": []string{"*.read"}},
			},
		},
		"application": map[string]any{
			"version":     "0.1.0-dev",
			"scriptsPath": cfg.ScriptsPath,
		},
	})
}

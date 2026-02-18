package handlers

import (
	"net/http"
)

func handleGetSettingsMock(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{
		"connection": map[string]any{
			"domainControllers": []map[string]any{
				{"hostname": "dc1.dzsec.net", "address": "10.10.1.10", "port": 636, "site": "Bozeman", "primary": true, "status": "connected"},
				{"hostname": "dc2.dzsec.net", "address": "10.10.1.11", "port": 636, "site": "Bozeman", "primary": false, "status": "connected"},
				{"hostname": "dc3.dzsec.net", "address": "10.20.1.10", "port": 636, "site": "Seattle", "primary": false, "status": "degraded"},
			},
			"baseDN":   "DC=dzsec,DC=net",
			"realm":    "DZSEC.NET",
			"protocol": "ldaps",
		},
		"tls": map[string]any{
			"provider":    "letsencrypt",
			"domain":      "sambmin.dzsec.net",
			"certificate":  "/usr/local/etc/letsencrypt/live/sambmin.dzsec.net/fullchain.pem",
			"key":          "/usr/local/etc/letsencrypt/live/sambmin.dzsec.net/privkey.pem",
			"expiry":       "2026-05-18T00:00:00Z",
			"autoRenew":    true,
		},
		"auth": map[string]any{
			"kerberos": map[string]any{
				"enabled":        true,
				"implementation": "heimdal",
				"keytab":         "/usr/local/etc/sambmin/sambmin.keytab",
				"spn":            "HTTP/sambmin.dzsec.net@DZSEC.NET",
			},
			"ldapBind": map[string]any{
				"enabled": true,
			},
			"sessionTimeout": 8,
		},
		"rbac": map[string]any{
			"roles": []map[string]any{
				{"role": "Full Admin", "groups": []string{"Domain Admins", "Enterprise Admins"}, "permissions": []string{"*"}},
				{"role": "User Admin", "groups": []string{"Account Operators"}, "permissions": []string{"users.*", "groups.read"}},
				{"role": "DNS Admin", "groups": []string{"DnsAdmins"}, "permissions": []string{"dns.*"}},
				{"role": "Help Desk", "groups": []string{"Help Desk"}, "permissions": []string{"users.read", "users.resetPassword", "users.unlock"}},
				{"role": "Read Only", "groups": []string{"Domain Users"}, "permissions": []string{"*.read"}},
			},
		},
		"application": map[string]any{
			"version":      "0.1.0-dev",
			"scriptsPath":  "/usr/local/share/sambmin/scripts",
			"databaseHost": "localhost:5432",
			"databaseName": "sambmin",
			"auditRetentionDays": 90,
		},
	})
}

package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"

	goldap "github.com/go-ldap/ldap/v3"
)

// handleKerberosPolicy returns Kerberos-related domain policy settings.
// GET /api/kerberos/policy
func handleKerberosPolicy(w http.ResponseWriter, r *http.Request) {
	if dirClient == nil {
		respondError(w, http.StatusServiceUnavailable, "directory not available")
		return
	}

	conn, err := dirClient.GetConn(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "ldap connection: "+err.Error())
		return
	}
	defer dirClient.PutConn(conn)

	baseDN := dirClient.BaseDN()

	// Read domain object for Kerberos-related attributes
	sr := goldap.NewSearchRequest(
		baseDN,
		goldap.ScopeBaseObject,
		goldap.NeverDerefAliases,
		0, 1, false,
		"(objectClass=domain)",
		[]string{
			"msDS-SupportedEncryptionTypes",
			"maxPwdAge", "minPwdAge", "minPwdLength",
			"pwdHistoryLength", "lockoutDuration", "lockoutThreshold",
			"lockOutObservationWindow",
		},
		nil,
	)

	result, err := conn.Search(sr)
	if err != nil {
		slog.Error("kerberos: domain search failed", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to read domain policy")
		return
	}

	policy := map[string]any{
		"realm":  strings.ToUpper(strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(baseDN, "DC="), "dc="), ",DC=", ".")),
		"baseDN": baseDN,
	}

	if len(result.Entries) > 0 {
		e := result.Entries[0]
		policy["maxPwdAge"] = e.GetAttributeValue("maxPwdAge")
		policy["minPwdAge"] = e.GetAttributeValue("minPwdAge")
		policy["minPwdLength"] = e.GetAttributeValue("minPwdLength")
		policy["pwdHistoryLength"] = e.GetAttributeValue("pwdHistoryLength")
		policy["lockoutDuration"] = e.GetAttributeValue("lockoutDuration")
		policy["lockoutThreshold"] = e.GetAttributeValue("lockoutThreshold")
		policy["lockoutObservationWindow"] = e.GetAttributeValue("lockOutObservationWindow")
		policy["supportedEncryptionTypes"] = e.GetAttributeValue("msDS-SupportedEncryptionTypes")
	}

	if handlerConfig != nil {
		policy["implementation"] = handlerConfig.Kerberos.Implementation
		policy["kdc"] = handlerConfig.Kerberos.KDC
		policy["keytabConfigured"] = handlerConfig.Kerberos.KeytabPath != ""
	}

	respondJSON(w, http.StatusOK, policy)
}

// handleKerberosAccounts returns accounts that have SPNs registered.
// GET /api/kerberos/accounts
func handleKerberosAccounts(w http.ResponseWriter, r *http.Request) {
	if dirClient == nil {
		respondError(w, http.StatusServiceUnavailable, "directory not available")
		return
	}

	conn, err := dirClient.GetConn(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "ldap connection: "+err.Error())
		return
	}
	defer dirClient.PutConn(conn)

	// Find all accounts with at least one SPN
	sr := goldap.NewSearchRequest(
		dirClient.BaseDN(),
		goldap.ScopeWholeSubtree,
		goldap.NeverDerefAliases,
		0, 0, false,
		"(servicePrincipalName=*)",
		[]string{"dn", "sAMAccountName", "displayName", "objectClass", "servicePrincipalName", "userAccountControl", "msDS-SupportedEncryptionTypes"},
		nil,
	)

	result, err := conn.SearchWithPaging(sr, 500)
	if err != nil {
		slog.Error("kerberos: accounts search failed", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to search accounts")
		return
	}

	type account struct {
		DN               string   `json:"dn"`
		SAMAccountName   string   `json:"samAccountName"`
		DisplayName      string   `json:"displayName"`
		ObjectType       string   `json:"objectType"`
		SPNs             []string `json:"spns"`
		SPNCount         int      `json:"spnCount"`
		EncryptionTypes  string   `json:"encryptionTypes,omitempty"`
	}

	var accounts []account
	for _, entry := range result.Entries {
		classes := entry.GetAttributeValues("objectClass")
		objType := "user"
		for _, cls := range classes {
			if cls == "computer" {
				objType = "computer"
				break
			}
		}

		spns := entry.GetAttributeValues("servicePrincipalName")
		name := entry.GetAttributeValue("displayName")
		if name == "" {
			name = entry.GetAttributeValue("sAMAccountName")
		}

		accounts = append(accounts, account{
			DN:              entry.DN,
			SAMAccountName:  entry.GetAttributeValue("sAMAccountName"),
			DisplayName:     name,
			ObjectType:      objType,
			SPNs:            spns,
			SPNCount:        len(spns),
			EncryptionTypes: entry.GetAttributeValue("msDS-SupportedEncryptionTypes"),
		})
	}

	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].SAMAccountName < accounts[j].SAMAccountName
	})

	respondJSON(w, http.StatusOK, map[string]any{
		"accounts": accounts,
		"total":    len(accounts),
	})
}

// handleExportKeytab exports a keytab for one or more principals.
// If the service lacks SAM access, returns CLI commands instead of a file.
// POST /api/kerberos/keytab
func handleExportKeytab(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req struct {
		Principal  string   `json:"principal"`  // single principal (backward compat)
		Principals []string `json:"principals"` // multiple principals
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Merge single + multi into one list, deduplicate
	seen := make(map[string]bool)
	var principals []string
	for _, p := range append(req.Principals, req.Principal) {
		p = strings.TrimSpace(p)
		if p != "" && !seen[p] {
			seen[p] = true
			principals = append(principals, p)
		}
	}
	if len(principals) == 0 {
		respondError(w, http.StatusBadRequest, "at least one principal required")
		return
	}

	// Export keytab — samba-tool appends to existing file, so call once per principal
	tmpFile := fmt.Sprintf("/tmp/sambmin-keytab-%s.keytab", sess.ID[:8])
	os.Remove(tmpFile) // clean any stale file

	var exported []string
	for _, principal := range principals {
		if _, err := runSambaTool(r.Context(), sess, "domain", "exportkeytab", tmpFile, "--principal="+principal); err != nil {
			errStr := err.Error()

			// Detect permission denied — service lacks SAM database access
			if strings.Contains(errStr, "Permission denied") || strings.Contains(errStr, "sam.ldb") {
				slog.Warn("keytab export: SAM access denied, returning CLI fallback", "actor", sess.Username)

				// Build CLI commands for the user to run manually
				var commands []string
				for i, p := range principals {
					if i == 0 {
						commands = append(commands, fmt.Sprintf("samba-tool domain exportkeytab /tmp/service.keytab --principal=%s", p))
					} else {
						commands = append(commands, fmt.Sprintf("samba-tool domain exportkeytab /tmp/service.keytab --principal=%s", p))
					}
				}

				respondJSON(w, http.StatusOK, map[string]any{
					"mode":     "cli_fallback",
					"message":  "Keytab export requires direct access to the Samba private database. Run these commands on the DC as root:",
					"commands": commands,
				})
				os.Remove(tmpFile)
				return
			}

			// Other errors (bad principal, etc.)
			slog.Error("keytab export failed", "principal", principal, "actor", sess.Username, "error", err)
			respondError(w, http.StatusInternalServerError, "keytab export failed for "+principal+": "+errStr)
			os.Remove(tmpFile)
			return
		}
		exported = append(exported, principal)
	}

	// Read and serve the file as download
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		slog.Error("keytab read failed", "file", tmpFile, "error", err)
		respondError(w, http.StatusInternalServerError, "keytab file read failed")
		return
	}
	os.Remove(tmpFile) // cleanup

	filename := "service.keytab"
	if len(exported) == 1 {
		filename = strings.ReplaceAll(exported[0], "/", "_") + ".keytab"
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(http.StatusOK)
	w.Write(data)

	slog.Info("keytab exported", "principals", exported, "count", len(exported), "actor", sess.Username)
	LogAudit(sess.Username, "kerberos.exportKeytab", "", "keytab", "", true, fmt.Sprintf("principals=%v", exported))
}

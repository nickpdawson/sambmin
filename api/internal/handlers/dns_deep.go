package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	dnspkg "github.com/nickdawson/sambmin/internal/dns"
	"github.com/nickdawson/sambmin/internal/models"
)

// dnsCmdTimeout is the timeout for DNS samba-tool commands.
const dnsCmdTimeout = 15 * time.Second

// runDNSCommand runs a samba-tool command using the service account credentials.
// For non-DNS commands (sites, fsmo, drs, gpo, spn, delegation), it adds
// -H ldap://localhost to avoid local sam.ldb permission errors.
// DNS commands use a positional server arg and don't need -H.
func runDNSCommand(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dnsCmdTimeout)
	defer cancel()

	username := usernameFromBindDN(handlerConfig.BindDN)
	password := os.Getenv("SAMBMIN_BIND_PW")
	if password == "" {
		password = handlerConfig.BindPW
	}

	// Non-DNS samba-tool commands need -H ldap://localhost to connect via
	// LDAP instead of opening the local sam.ldb file (which requires root).
	// Exception: "drs" commands use DCE/RPC, not LDAP, and don't accept -H.
	if len(args) > 0 && args[0] != "dns" && args[0] != "drs" {
		args = append(args, "-H", "ldap://localhost")
	}

	if username != "" && password != "" {
		args = append(args, "-U", fmt.Sprintf("%s%%%s", username, password))
	}

	cmd := exec.CommandContext(ctx, sambaTool, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	slog.Debug("dns: running samba-tool", "args", args[:minInt2(2, len(args))])

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		// Strip samba-tool warnings, Python traceback noise, and extract meaningful error.
		lines := strings.Split(errMsg, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line == "" || strings.HasPrefix(line, "WARNING:") || strings.HasPrefix(line, "Usage:") || strings.Contains(line, "setproctitle") {
				continue
			}
			if strings.HasPrefix(line, "File ") || strings.HasPrefix(line, "^^^^") || strings.Trim(line, "^ ") == "" {
				continue
			}
			errMsg = line
			break
		}
		return "", fmt.Errorf("samba-tool %s: %s",
			strings.Join(args[:minInt2(2, len(args))], " "), errMsg)
	}

	return stdout.String(), nil
}

// runDNSCommandForHost is like runDNSCommand but connects to a specific DC
// via -H ldap://<host> instead of the default localhost.
func runDNSCommandForHost(host string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dnsCmdTimeout)
	defer cancel()

	username := usernameFromBindDN(handlerConfig.BindDN)
	password := os.Getenv("SAMBMIN_BIND_PW")
	if password == "" {
		password = handlerConfig.BindPW
	}

	// Non-DNS commands need -H to specify the target DC.
	// Exception: "drs" commands use DCE/RPC and take the DC as a positional arg.
	if len(args) > 0 && args[0] == "drs" {
		// Insert host after the subcommand: drs showrepl <host>
		if len(args) >= 2 {
			args = append(args[:2], append([]string{host}, args[2:]...)...)
		}
	} else if len(args) > 0 && args[0] != "dns" {
		args = append(args, "-H", fmt.Sprintf("ldap://%s", host))
	}

	if username != "" && password != "" {
		args = append(args, "-U", fmt.Sprintf("%s%%%s", username, password))
	}

	cmd := exec.CommandContext(ctx, sambaTool, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	slog.Debug("dns: running samba-tool", "host", host, "args", args[:minInt2(2, len(args))])

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		lines := strings.Split(errMsg, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line == "" || strings.HasPrefix(line, "WARNING:") || strings.HasPrefix(line, "Usage:") || strings.Contains(line, "setproctitle") {
				continue
			}
			if strings.HasPrefix(line, "File ") || strings.HasPrefix(line, "^^^^") || strings.Trim(line, "^ ") == "" {
				continue
			}
			errMsg = line
			break
		}
		return "", fmt.Errorf("samba-tool %s: %s",
			strings.Join(args[:minInt2(2, len(args))], " "), errMsg)
	}

	return stdout.String(), nil
}

func minInt2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// handleDNSServerInfo returns DNS server configuration.
// GET /api/dns/serverinfo
func handleDNSServerInfo(w http.ResponseWriter, _ *http.Request) {
	server := primaryDCHostname()
	output, err := runDNSCommand("dns", "serverinfo", server)
	if err != nil {
		slog.Error("dns: failed to get server info", "error", err)
		respondError(w, http.StatusInternalServerError,
			"failed to get DNS server info: "+err.Error())
		return
	}

	info := dnspkg.ParseServerInfo(output, server)
	respondJSON(w, http.StatusOK, info)
}

// handleDNSZoneInfo returns detailed zone properties including aging/scavenging.
// GET /api/dns/zones/{zone}/info
func handleDNSZoneInfo(w http.ResponseWriter, r *http.Request) {
	zone := r.PathValue("zone")
	if zone == "" {
		respondError(w, http.StatusBadRequest, "zone name required")
		return
	}

	output, err := runDNSCommand("dns", "zoneinfo", primaryDCHostname(), zone)
	if err != nil {
		slog.Error("dns: failed to get zone info", "zone", zone, "error", err)
		respondError(w, http.StatusInternalServerError,
			fmt.Sprintf("failed to get zone info for %s: %s", zone, err.Error()))
		return
	}

	info := dnspkg.ParseZoneInfo(output, zone)
	respondJSON(w, http.StatusOK, info)
}

// handleDNSZoneOptions updates zone aging/scavenging options.
// PUT /api/dns/zones/{zone}/options
func handleDNSZoneOptions(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	zone := r.PathValue("zone")
	if zone == "" {
		respondError(w, http.StatusBadRequest, "zone name required")
		return
	}

	var req struct {
		Aging             *bool `json:"aging"`
		NoRefreshInterval *int  `json:"noRefreshInterval"` // hours
		RefreshInterval   *int  `json:"refreshInterval"`   // hours
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	server := primaryDCHostname()

	if req.Aging != nil {
		val := "0"
		if *req.Aging {
			val = "1"
		}
		if _, err := runSambaTool(r.Context(), sess, "dns", "zoneoptions", server, zone, "--aging="+val); err != nil {
			slog.Error("DNS zone options update failed", "zone", zone, "option", "aging", "actor", sess.Username, "error", err)
			respondSafeError(w, http.StatusInternalServerError, "DNS zone aging update failed", err)
			return
		}
	}

	if req.NoRefreshInterval != nil {
		val := fmt.Sprintf("%d", *req.NoRefreshInterval)
		if _, err := runSambaTool(r.Context(), sess, "dns", "zoneoptions", server, zone, "--norefreshinterval="+val); err != nil {
			slog.Error("DNS zone options update failed", "zone", zone, "option", "norefreshinterval", "actor", sess.Username, "error", err)
			respondSafeError(w, http.StatusInternalServerError, "DNS zone no-refresh interval update failed", err)
			return
		}
	}

	if req.RefreshInterval != nil {
		val := fmt.Sprintf("%d", *req.RefreshInterval)
		if _, err := runSambaTool(r.Context(), sess, "dns", "zoneoptions", server, zone, "--refreshinterval="+val); err != nil {
			slog.Error("DNS zone options update failed", "zone", zone, "option", "refreshinterval", "actor", sess.Username, "error", err)
			respondSafeError(w, http.StatusInternalServerError, "DNS zone refresh interval update failed", err)
			return
		}
	}

	slog.Info("DNS zone options updated", "zone", zone, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "zone": zone})
}

// handleDNSQuery queries DNS records from a specific DC.
// POST /api/dns/query
func handleDNSQuery(w http.ResponseWriter, r *http.Request) {
	var req models.DNSQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Zone == "" || req.Name == "" {
		respondError(w, http.StatusBadRequest, "zone and name required")
		return
	}
	if req.Server == "" {
		req.Server = primaryDCHostname()
	}
	if req.Type == "" {
		req.Type = "ALL"
	}

	output, err := runDNSCommand("dns", "query", req.Server, req.Zone, req.Name, req.Type)
	result := models.DNSQueryResult{
		Server: req.Server,
		Zone:   req.Zone,
		Name:   req.Name,
	}
	if err != nil {
		if strings.Contains(err.Error(), "WERR_DNS_ERROR_NAME_DOES_NOT_EXIST") {
			result.Records = []models.DNSRecord{}
		} else {
			result.Error = err.Error()
			result.Records = []models.DNSRecord{}
		}
	} else {
		result.Records = dnspkg.ParseRecordOutput(output, req.Name)
	}

	respondJSON(w, http.StatusOK, result)
}

// handleDNSSRVValidator performs per-site/per-DC SRV record validation.
// GET /api/dns/srv-validator
func handleDNSSRVValidator(w http.ResponseWriter, _ *http.Request) {
	domainZone := domainFromBaseDN(handlerConfig.BaseDN)

	srvRecords := []string{
		"_ldap._tcp",
		"_kerberos._tcp",
		"_gc._tcp",
		"_ldap._tcp.dc._msdcs",
		"_kerberos._tcp.dc._msdcs",
		"_kpasswd._tcp",
		"_ldap._tcp.pdc._msdcs",
	}

	dcNames := []string{}
	for _, dc := range handlerConfig.DCs {
		dcNames = append(dcNames, dc.Hostname)
	}
	if len(dcNames) == 0 {
		dcNames = []string{primaryDCHostname()}
	}

	var entries []models.SRVValidationEntry
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, record := range srvRecords {
		for _, dc := range dcNames {
			wg.Add(1)
			go func(rec, dcName string) {
				defer wg.Done()

				entry := models.SRVValidationEntry{
					Record: rec,
					DC:     dcName,
				}

				server := dcName
				for _, dcCfg := range handlerConfig.DCs {
					if dcCfg.Hostname == dcName {
						server = dcCfg.Address
						break
					}
				}

				output, err := runDNSCommand("dns", "query", server, domainZone, rec, "SRV")
				if err != nil {
					if strings.Contains(err.Error(), "WERR_DNS_ERROR_NAME_DOES_NOT_EXIST") {
						entry.Status = "fail"
						entry.Message = "No SRV records found"
					} else {
						entry.Status = "error"
						entry.Message = err.Error()
					}
				} else {
					records := dnspkg.ParseRecordOutput(output, rec)
					if len(records) == 0 {
						entry.Status = "fail"
						entry.Message = "No SRV records found"
					} else {
						entry.Status = "pass"
						entry.Targets = len(records)
						var targets []string
						for _, r := range records {
							targets = append(targets, r.Value)
						}
						entry.Message = strings.Join(targets, ", ")
					}
				}

				mu.Lock()
				entries = append(entries, entry)
				mu.Unlock()
			}(record, dc)
		}
	}

	wg.Wait()

	total := len(entries)
	passed := 0
	failed := 0
	errCount := 0
	for _, e := range entries {
		switch e.Status {
		case "pass":
			passed++
		case "fail":
			failed++
		case "error":
			errCount++
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"entries": entries,
		"records": srvRecords,
		"dcs":     dcNames,
		"summary": map[string]int{
			"total":  total,
			"passed": passed,
			"failed": failed,
			"errors": errCount,
		},
	})
}

// handleDNSConsistency checks DNS consistency across DCs.
// GET /api/dns/consistency
func handleDNSConsistency(w http.ResponseWriter, r *http.Request) {
	zone := r.URL.Query().Get("zone")
	if zone == "" {
		zone = domainFromBaseDN(handlerConfig.BaseDN)
	}

	type dcEntry struct {
		Name    string
		Address string
	}

	dcs := []dcEntry{}
	for _, dc := range handlerConfig.DCs {
		dcs = append(dcs, dcEntry{dc.Hostname, dc.Address})
	}
	if len(dcs) == 0 {
		dcs = append(dcs, dcEntry{primaryDCHostname(), "127.0.0.1"})
	}

	type dcResult struct {
		DC        string `json:"dc"`
		SOASerial uint32 `json:"soaSerial"`
		Records   int    `json:"records"`
		Status    string `json:"status"`
		Error     string `json:"error,omitempty"`
	}

	var results []dcResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, dc := range dcs {
		wg.Add(1)
		go func(dcName, dcAddr string) {
			defer wg.Done()

			result := dcResult{DC: dcName}

			output, err := runDNSCommand("dns", "query", dcAddr, zone, "@", "SOA")
			if err != nil {
				result.Status = "error"
				result.Error = err.Error()
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}

			records := dnspkg.ParseRecordOutput(output, "@")
			for _, rec := range records {
				if rec.Type == "SOA" {
					fields := strings.Fields(rec.Value)
					if len(fields) >= 3 {
						fmt.Sscanf(fields[2], "%d", &result.SOASerial)
					}
				}
			}

			allOutput, err := runDNSCommand("dns", "query", dcAddr, zone, "*", "ALL")
			if err != nil {
				result.Records = -1
			} else {
				allRecs := dnspkg.ParseRecordOutput(allOutput, "")
				result.Records = len(allRecs)
			}

			result.Status = "ok"

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(dc.Name, dc.Address)
	}

	wg.Wait()

	consistent := true
	if len(results) > 1 {
		var firstSerial uint32
		foundFirst := false
		for _, r := range results {
			if r.Status == "error" {
				continue
			}
			if !foundFirst {
				firstSerial = r.SOASerial
				foundFirst = true
			} else if r.SOASerial != firstSerial {
				consistent = false
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"zone":       zone,
		"consistent": consistent,
		"dcs":        results,
	})
}

// handleDNSLimitations returns known Samba DNS limitations.
// GET /api/dns/limitations
func handleDNSLimitations(w http.ResponseWriter, _ *http.Request) {
	limitations := []map[string]string{
		{
			"id":          "conditional-forwarders",
			"title":       "Conditional Forwarders",
			"description": "Samba DNS does not support conditional forwarders. Use BIND9 DLZ backend for this feature.",
			"severity":    "warning",
		},
		{
			"id":          "zone-transfers",
			"title":       "Zone Transfers (AXFR/IXFR)",
			"description": "Samba internal DNS does not support zone transfers. Replication uses AD replication instead.",
			"severity":    "info",
		},
		{
			"id":          "forwarder-config",
			"title":       "Forwarder Configuration",
			"description": "Forwarder changes require editing smb.conf and restarting the DNS service. They cannot be changed via samba-tool.",
			"severity":    "warning",
		},
		{
			"id":          "dnssec",
			"title":       "DNSSEC",
			"description": "Samba DNS does not support DNSSEC signing or validation.",
			"severity":    "info",
		},
		{
			"id":          "scavenging",
			"title":       "DNS Scavenging",
			"description": "Aging and scavenging configuration is supported, but automatic scavenging requires running samba_dnsupdate periodically.",
			"severity":    "info",
		},
	}

	respondJSON(w, http.StatusOK, map[string]any{"limitations": limitations})
}

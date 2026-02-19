package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nickdawson/sambmin/internal/models"
)

// ──────────────────────────────────────────────────────────────────────────────
// Replication
// ──────────────────────────────────────────────────────────────────────────────

// handleReplicationTopology returns the replication topology with links between DCs.
// GET /api/replication/topology
func handleReplicationTopologyLive(w http.ResponseWriter, _ *http.Request) {
	output, err := runDNSCommand("drs", "showrepl", "--json")
	if err != nil {
		slog.Warn("replication: showrepl --json failed, falling back to text", "error", err)
		// Fallback to text output
		output, err = runDNSCommand("drs", "showrepl")
		if err != nil {
			slog.Error("replication: showrepl failed", "error", err)
			respondError(w, http.StatusInternalServerError, "failed to get replication status: "+err.Error())
			return
		}
		topo := parseShowreplText(output)
		respondJSON(w, http.StatusOK, topo)
		return
	}

	// Try to parse JSON output from samba-tool drs showrepl --json
	var jsonResult map[string]any
	if err := json.Unmarshal([]byte(output), &jsonResult); err != nil {
		// Fall back to text parsing if JSON parse fails
		topo := parseShowreplText(output)
		respondJSON(w, http.StatusOK, topo)
		return
	}

	// Extract from JSON format
	topo := parseShowreplJSON(jsonResult)
	respondJSON(w, http.StatusOK, topo)
}

// handleReplicationStatus returns per-DC replication status summary.
// GET /api/replication/status
func handleReplicationStatusLive(w http.ResponseWriter, _ *http.Request) {
	type dcReplStatus struct {
		DC             string    `json:"dc"`
		Address        string    `json:"address"`
		Site           string    `json:"site"`
		Reachable      bool      `json:"reachable"`
		InboundOK      int       `json:"inboundOk"`
		InboundFailed  int       `json:"inboundFailed"`
		OutboundOK     int       `json:"outboundOk"`
		OutboundFailed int       `json:"outboundFailed"`
		LastSuccess    time.Time `json:"lastSuccess"`
		Error          string    `json:"error,omitempty"`
	}

	var results []dcReplStatus
	var mu sync.Mutex
	var wg sync.WaitGroup

	dcs := handlerConfig.DCs
	if len(dcs) == 0 {
		respondJSON(w, http.StatusOK, map[string]any{
			"dcs":     []dcReplStatus{},
			"summary": map[string]int{"total": 0, "healthy": 0, "degraded": 0, "failed": 0},
		})
		return
	}

	for _, dc := range dcs {
		wg.Add(1)
		go func(dc struct {
			Hostname string
			Address  string
			Site     string
		}) {
			defer wg.Done()
			r := dcReplStatus{
				DC:      dc.Hostname,
				Address: dc.Address,
				Site:    dc.Site,
			}

			output, err := runDNSCommand("drs", "showrepl", dc.Address)
			if err != nil {
				r.Error = err.Error()
				mu.Lock()
				results = append(results, r)
				mu.Unlock()
				return
			}

			r.Reachable = true
			in, out := countReplPartners(output)
			r.InboundOK = in
			r.OutboundOK = out
			r.LastSuccess = time.Now() // Placeholder — would parse from output

			mu.Lock()
			results = append(results, r)
			mu.Unlock()
		}(struct {
			Hostname string
			Address  string
			Site     string
		}{dc.Hostname, dc.Address, dc.Site})
	}

	wg.Wait()

	healthy, degraded, failed := 0, 0, 0
	for _, r := range results {
		switch {
		case !r.Reachable:
			failed++
		case r.InboundFailed > 0 || r.OutboundFailed > 0:
			degraded++
		default:
			healthy++
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"dcs": results,
		"summary": map[string]int{
			"total":    len(results),
			"healthy":  healthy,
			"degraded": degraded,
			"failed":   failed,
		},
	})
}

// handleForceSync triggers a replication sync via samba-tool drs replicate.
// POST /api/replication/sync
func handleForceSyncLive(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req struct {
		SourceDC string `json:"sourceDC"`
		DestDC   string `json:"destDC"`
		NC       string `json:"namingContext"` // optional, defaults to domain NC
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.SourceDC == "" || req.DestDC == "" {
		respondError(w, http.StatusBadRequest, "sourceDC and destDC required")
		return
	}

	nc := req.NC
	if nc == "" {
		nc = handlerConfig.BaseDN
	}

	// samba-tool drs replicate <dest> <source> <NC>
	output, err := runSambaTool(r.Context(), sess, "drs", "replicate", req.DestDC, req.SourceDC, nc)
	if err != nil {
		slog.Error("force sync failed", "src", req.SourceDC, "dest", req.DestDC, "nc", nc, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("force sync triggered", "src", req.SourceDC, "dest", req.DestDC, "nc", nc, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"output":  strings.TrimSpace(output),
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Sites & Subnets
// ──────────────────────────────────────────────────────────────────────────────

// handleListSitesLive returns AD sites via samba-tool sites list.
// GET /api/sites
func handleListSitesLive(w http.ResponseWriter, _ *http.Request) {
	output, err := runDNSCommand("sites", "list")
	if err != nil {
		slog.Error("sites: list failed", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list sites: "+err.Error())
		return
	}

	sites := parseSitesList(output)

	// Enrich with DC info from config
	for i := range sites {
		for _, dc := range handlerConfig.DCs {
			if dc.Site == sites[i].Name {
				sites[i].DCs = append(sites[i].DCs, dc.Hostname)
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{"sites": sites})
}

// handleCreateSiteLive creates a new AD site.
// POST /api/sites
func handleCreateSiteLive(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "site name required")
		return
	}

	if _, err := runSambaTool(r.Context(), sess, "sites", "create", req.Name); err != nil {
		slog.Error("site create failed", "name", req.Name, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("site created", "name", req.Name, "actor", sess.Username)
	respondJSON(w, http.StatusCreated, map[string]any{"success": true, "name": req.Name})
}

// handleListSubnetsLive returns subnets for a site.
// GET /api/sites/{site}/subnets
func handleListSubnetsLive(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	if site == "" {
		respondError(w, http.StatusBadRequest, "site name required")
		return
	}

	output, err := runDNSCommand("sites", "subnet", "list", "--site="+site)
	if err != nil {
		// Try without --site flag (some samba-tool versions list all then filter)
		output, err = runDNSCommand("sites", "subnet", "list")
		if err != nil {
			slog.Error("subnets: list failed", "site", site, "error", err)
			respondError(w, http.StatusInternalServerError, "failed to list subnets: "+err.Error())
			return
		}
	}

	subnets := parseSubnetsList(output, site)
	respondJSON(w, http.StatusOK, map[string]any{"subnets": subnets, "site": site})
}

// ──────────────────────────────────────────────────────────────────────────────
// FSMO Roles
// ──────────────────────────────────────────────────────────────────────────────

// handleGetFSMORolesLive returns the current FSMO role holders.
// GET /api/fsmo
func handleGetFSMORolesLive(w http.ResponseWriter, _ *http.Request) {
	output, err := runDNSCommand("fsmo", "show")
	if err != nil {
		slog.Error("fsmo: show failed", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get FSMO roles: "+err.Error())
		return
	}

	roles := parseFSMORoles(output)
	respondJSON(w, http.StatusOK, map[string]any{"roles": roles})
}

// handleTransferFSMOLive transfers a FSMO role.
// POST /api/fsmo/transfer
func handleTransferFSMOLive(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req struct {
		Role   string `json:"role"`   // "schema", "naming", "pdc", "rid", "infrastructure"
		Target string `json:"target"` // target DC
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Role == "" {
		respondError(w, http.StatusBadRequest, "role required")
		return
	}

	args := []string{"fsmo", "transfer", "--role=" + req.Role}

	output, err := runSambaTool(r.Context(), sess, args...)
	if err != nil {
		slog.Error("FSMO transfer failed", "role", req.Role, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("FSMO role transferred", "role", req.Role, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"role":    req.Role,
		"output":  strings.TrimSpace(output),
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Parsers
// ──────────────────────────────────────────────────────────────────────────────

// parseShowreplText parses the text output of samba-tool drs showrepl.
func parseShowreplText(output string) map[string]any {
	links := []models.ReplicationLink{}
	dcs := map[string]bool{}

	// Parse "Inbound Neighbors" and "Outbound Neighbors" sections
	sections := strings.Split(output, "==========")

	for _, section := range sections {
		lines := strings.Split(section, "\n")
		isInbound := false
		isOutbound := false

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "Inbound Neighbors") {
				isInbound = true
				isOutbound = false
			} else if strings.Contains(line, "Outbound Neighbors") {
				isOutbound = true
				isInbound = false
			}

			if (isInbound || isOutbound) && strings.Contains(line, "DSA objectGUID") {
				// Extract DC name from DSA objectGUID line
				// Format: "DSA objectGUID: <guid>\n\t\tDSA invocationId: ...\n\t\t\t<DC-name>"
			}

			// Parse naming context and source DC
			if strings.HasPrefix(line, "DC=") || strings.HasPrefix(line, "CN=") {
				ncMatch := line
				// Next relevant line should have source/dest info
				link := models.ReplicationLink{
					NamingContext: ncMatch,
					Status:       "current",
					LastSync:      time.Now(),
				}

				if isInbound {
					link.SourceDC = extractDCFromShowrepl(lines, line)
					link.DestDC = "localhost"
				} else if isOutbound {
					link.SourceDC = "localhost"
					link.DestDC = extractDCFromShowrepl(lines, line)
				}

				if link.SourceDC != "" || link.DestDC != "" {
					links = append(links, link)
					dcs[link.SourceDC] = true
					dcs[link.DestDC] = true
				}
			}

			// Detect failures
			if strings.Contains(line, "WERR_") || strings.Contains(line, "failed") {
				if len(links) > 0 {
					links[len(links)-1].Status = "failed"
				}
			}
		}
	}

	dcList := make([]string, 0, len(dcs))
	for dc := range dcs {
		if dc != "" {
			dcList = append(dcList, dc)
		}
	}

	return map[string]any{
		"links": links,
		"dcs":   dcList,
	}
}

// parseShowreplJSON parses the JSON output from samba-tool drs showrepl --json.
func parseShowreplJSON(data map[string]any) map[string]any {
	links := []models.ReplicationLink{}
	dcs := map[string]bool{}

	// JSON format has "repsFrom" and "repsTo" arrays
	for _, section := range []string{"repsFrom", "repsTo"} {
		entries, ok := data[section].([]any)
		if !ok {
			continue
		}
		for _, entry := range entries {
			e, ok := entry.(map[string]any)
			if !ok {
				continue
			}

			link := models.ReplicationLink{
				Status:   "current",
				LastSync: time.Now(),
			}

			if nc, ok := e["naming context"].(string); ok {
				link.NamingContext = nc
			}
			if dc, ok := e["DSA"].(string); ok {
				if section == "repsFrom" {
					link.SourceDC = dc
					link.DestDC = "localhost"
				} else {
					link.SourceDC = "localhost"
					link.DestDC = dc
				}
			}

			// Check last attempt/result
			if result, ok := e["last attempt message"].(string); ok {
				if strings.Contains(result, "successful") || result == "WERR_OK" {
					link.Status = "current"
				} else {
					link.Status = "failed"
				}
			}

			if link.SourceDC != "" || link.DestDC != "" {
				links = append(links, link)
				dcs[link.SourceDC] = true
				dcs[link.DestDC] = true
			}
		}
	}

	dcList := make([]string, 0, len(dcs))
	for dc := range dcs {
		if dc != "" {
			dcList = append(dcList, dc)
		}
	}

	return map[string]any{
		"links": links,
		"dcs":   dcList,
	}
}

// extractDCFromShowrepl tries to extract a DC hostname from nearby showrepl text.
func extractDCFromShowrepl(lines []string, currentLine string) string {
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for "DSA objectGUID" followed by hostname-like strings
		if strings.Contains(line, "via RPC") {
			// Extract DC name from "via RPC, objectGUID ..." or similar
			parts := strings.Fields(line)
			for _, p := range parts {
				if strings.Contains(p, ".") && !strings.HasPrefix(p, "CN=") {
					return strings.TrimSuffix(p, ",")
				}
			}
		}
	}
	_ = currentLine
	return ""
}

// countReplPartners counts successful inbound and outbound replication partners
// from samba-tool drs showrepl output.
func countReplPartners(output string) (inbound, outbound int) {
	lines := strings.Split(output, "\n")
	section := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Inbound Neighbors") {
			section = "in"
		} else if strings.Contains(line, "Outbound Neighbors") {
			section = "out"
		}
		if strings.Contains(line, "successful") || strings.Contains(line, "WERR_OK") {
			switch section {
			case "in":
				inbound++
			case "out":
				outbound++
			}
		}
	}
	return
}

// parseSitesList parses samba-tool sites list output.
// Format: one site name per line.
func parseSitesList(output string) []models.Site {
	var sites []models.Site
	for _, line := range strings.Split(output, "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		sites = append(sites, models.Site{
			Name: name,
		})
	}
	return sites
}

// parseSubnetsList parses samba-tool sites subnet list output.
// Format varies — may be "subnet: site" or just subnet per line.
func parseSubnetsList(output, filterSite string) []string {
	var subnets []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// If output has "subnet, site" format
		if strings.Contains(line, ",") {
			parts := strings.SplitN(line, ",", 2)
			subnet := strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				site := strings.TrimSpace(parts[1])
				if filterSite != "" && !strings.EqualFold(site, filterSite) {
					continue
				}
			}
			subnets = append(subnets, subnet)
		} else {
			subnets = append(subnets, line)
		}
	}
	return subnets
}

// fsmoRoleEntry represents a single FSMO role holder.
type fsmoRoleEntry struct {
	Role   string `json:"role"`
	Holder string `json:"holder"`
	DC     string `json:"dc"` // extracted hostname
}

// parseFSMORoles parses samba-tool fsmo show output.
// Format: "RoleName owner: CN=NTDS Settings,CN=DC1,CN=Servers,CN=Site,CN=Sites,CN=Configuration,DC=..."
func parseFSMORoles(output string) []fsmoRoleEntry {
	var roles []fsmoRoleEntry

	roleNames := map[string]string{
		"SchemaMasterRole":         "Schema Master",
		"InfrastructureMasterRole": "Infrastructure Master",
		"RidAllocationMasterRole":  "RID Master",
		"PdcEmulationMasterRole":   "PDC Emulator",
		"DomainNamingMasterRole":   "Domain Naming Master",
		"DomainDnsZonesMasterRole": "Domain DNS Zones Master",
		"ForestDnsZonesMasterRole": "Forest DNS Zones Master",
	}

	dcPattern := regexp.MustCompile(`CN=([^,]+),CN=Servers`)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " owner: ", 2)
		if len(parts) != 2 {
			continue
		}

		roleName := strings.TrimSpace(parts[0])
		holder := strings.TrimSpace(parts[1])

		displayName := roleName
		if mapped, ok := roleNames[roleName]; ok {
			displayName = mapped
		}

		dc := ""
		if matches := dcPattern.FindStringSubmatch(holder); len(matches) > 1 {
			dc = matches[1]
		}

		roles = append(roles, fsmoRoleEntry{
			Role:   displayName,
			Holder: holder,
			DC:     dc,
		})
	}

	return roles
}

// ──────────────────────────────────────────────────────────────────────────────
// Audit Log (placeholder with in-memory store, PostgreSQL in M20)
// ──────────────────────────────────────────────────────────────────────────────

// auditEntries is an in-memory audit log store. Replaced with PostgreSQL in M20.
var auditEntries []models.AuditEntry
var auditMu sync.Mutex
var auditNextID int64 = 1

// handleListAuditLogLive returns audit log entries.
// GET /api/audit
func handleListAuditLogLive(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	auditMu.Lock()
	entries := make([]models.AuditEntry, 0)
	start := len(auditEntries) - limit
	if start < 0 {
		start = 0
	}
	entries = append(entries, auditEntries[start:]...)
	auditMu.Unlock()

	// Reverse so newest first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"entries": entries,
		"total":   len(auditEntries),
	})
}

// LogAudit adds an entry to the in-memory audit log.
func LogAudit(actor, action, objectDN, objectType, dc string, success bool, details string) {
	auditMu.Lock()
	defer auditMu.Unlock()

	auditEntries = append(auditEntries, models.AuditEntry{
		ID:         auditNextID,
		Timestamp:  time.Now(),
		Actor:      actor,
		Action:     action,
		ObjectDN:   objectDN,
		ObjectType: objectType,
		DC:         dc,
		Success:    success,
		Details:    details,
	})
	auditNextID++

	// Keep max 10000 entries in memory
	if len(auditEntries) > 10000 {
		auditEntries = auditEntries[len(auditEntries)-10000:]
	}
}

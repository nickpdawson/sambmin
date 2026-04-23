package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/nickdawson/sambmin/internal/dns"
)

// dnsClient is the shared samba-tool DNS client, initialized lazily on first use.
var dnsClient *dns.SambaClient

// getDNSClient returns (or lazily creates) the shared SambaClient.
// It extracts the service account username from the bind DN configured in
// handlerConfig and reads the password from SAMBMIN_BIND_PW or config.
func getDNSClient() *dns.SambaClient {
	if dnsClient != nil {
		return dnsClient
	}

	server := primaryDCHostname()
	username := usernameFromBindDN(handlerConfig.BindDN)
	password := os.Getenv("SAMBMIN_BIND_PW")
	if password == "" {
		password = handlerConfig.BindPW
	}

	dnsClient = dns.NewSambaClient(server, username, password)
	slog.Info("dns: initialized samba-tool client", "server", server, "user", username)
	return dnsClient
}

// usernameFromBindDN extracts the CN value from a bind DN.
// "CN=services,CN=Users,DC=example,DC=com" -> "services"
func usernameFromBindDN(bindDN string) string {
	if bindDN == "" {
		return ""
	}
	parts := strings.Split(bindDN, ",")
	if len(parts) == 0 {
		return ""
	}
	first := parts[0]
	if strings.HasPrefix(strings.ToUpper(first), "CN=") {
		return first[3:]
	}
	return first
}

// handleListDNSZonesLive queries DNS zones from the DC via samba-tool.
func handleListDNSZonesLive(w http.ResponseWriter, r *http.Request) {
	client := getDNSClient()

	zones, err := client.ListZones(r.Context())
	if err != nil {
		slog.Error("dns: failed to list zones", "error", err)
		respondError(w, http.StatusInternalServerError,
			"failed to list DNS zones: "+err.Error())
		return
	}

	// Enrich zones with record counts by querying each zone
	for i := range zones {
		records, err := client.QueryAllRecords(r.Context(), zones[i].Name)
		if err != nil {
			slog.Debug("dns: failed to count records for zone",
				"zone", zones[i].Name, "error", err)
			continue
		}
		zones[i].Records = len(records)

		// Extract SOA serial if a SOA record is present
		for _, rec := range records {
			if rec.Type == "SOA" {
				// SOA value format: "ns email serial refresh retry expire minttl"
				fields := strings.Fields(rec.Value)
				if len(fields) >= 3 {
					fmt.Sscanf(fields[2], "%d", &zones[i].SOASerial)
				}
				break
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"zones": zones,
		"total": len(zones),
	})
}

// handleListDNSRecordsLive queries DNS records for a specific zone.
func handleListDNSRecordsLive(w http.ResponseWriter, r *http.Request) {
	zone := r.PathValue("zone")
	if zone == "" {
		respondError(w, http.StatusBadRequest, "zone name is required")
		return
	}

	client := getDNSClient()

	records, err := client.QueryAllRecords(r.Context(), zone)
	if err != nil {
		slog.Error("dns: failed to list records", "zone", zone, "error", err)
		respondError(w, http.StatusInternalServerError,
			fmt.Sprintf("failed to list DNS records for zone %s: %s", zone, err.Error()))
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"zone":    zone,
		"records": records,
		"total":   len(records),
	})
}

// handleDNSDiagnosticsLive checks essential AD DNS SRV records.
func handleDNSDiagnosticsLive(w http.ResponseWriter, r *http.Request) {
	client := getDNSClient()

	// Determine the primary domain zone from config
	domainZone := domainFromBaseDN(handlerConfig.BaseDN)

	checks := []map[string]any{}

	// Check essential AD SRV records
	srvChecks := []struct {
		name    string
		record  string
		zone    string
		desc    string
	}{
		{"_ldap._tcp SRV", "_ldap._tcp", domainZone, "LDAP service location"},
		{"_kerberos._tcp SRV", "_kerberos._tcp", domainZone, "Kerberos KDC service location"},
		{"_gc._tcp SRV", "_gc._tcp", domainZone, "Global Catalog service location"},
		{"_ldap._tcp.dc._msdcs SRV", "_ldap._tcp.dc._msdcs", domainZone, "DC locator for LDAP"},
		{"_kerberos._tcp.dc._msdcs SRV", "_kerberos._tcp.dc._msdcs", domainZone, "DC locator for Kerberos"},
	}

	for _, chk := range srvChecks {
		check := runSRVCheck(r.Context(), client, chk.name, chk.record, chk.zone, chk.desc)
		checks = append(checks, check)
	}

	// Summary check for required SRV records
	allPass := true
	var failedNames []string
	for _, c := range checks {
		if c["status"] != "pass" {
			allPass = false
			failedNames = append(failedNames, c["name"].(string))
		}
	}

	summary := map[string]any{
		"name":   "AD SRV Records",
		"status": "pass",
	}
	if allPass {
		summary["message"] = "All required SRV records present (_ldap._tcp, _kerberos._tcp, _gc._tcp)"
	} else {
		summary["status"] = "warning"
		summary["message"] = fmt.Sprintf("Missing SRV records: %s", strings.Join(failedNames, ", "))
	}

	// Prepend summary, then individual checks as details
	result := []map[string]any{summary}
	result = append(result, checks...)

	respondJSON(w, http.StatusOK, map[string]any{
		"checks": result,
	})
}

// runSRVCheck queries a specific SRV record and returns a diagnostic check result.
func runSRVCheck(ctx context.Context, client *dns.SambaClient, name, record, zone, desc string) map[string]any {
	records, err := client.GetZoneRecords(ctx, zone, record)
	if err != nil {
		return map[string]any{
			"name":    name,
			"status":  "error",
			"message": fmt.Sprintf("Failed to query %s: %s", record, err.Error()),
		}
	}

	if len(records) == 0 {
		return map[string]any{
			"name":    name,
			"status":  "warning",
			"message": fmt.Sprintf("No %s records found for %s (%s)", record, zone, desc),
		}
	}

	// Build a summary of the targets found
	var targets []string
	for _, r := range records {
		targets = append(targets, r.Value)
	}

	return map[string]any{
		"name":    name,
		"status":  "pass",
		"message": fmt.Sprintf("%d record(s): %s", len(records), strings.Join(targets, ", ")),
	}
}

// domainFromBaseDN converts a base DN to a domain name.
// "DC=example,DC=com" -> "example.com"
func domainFromBaseDN(baseDN string) string {
	var parts []string
	for _, component := range strings.Split(baseDN, ",") {
		component = strings.TrimSpace(component)
		if strings.HasPrefix(strings.ToUpper(component), "DC=") {
			parts = append(parts, component[3:])
		}
	}
	return strings.Join(parts, ".")
}

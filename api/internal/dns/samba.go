package dns

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nickdawson/sambmin/internal/models"
)

// sambaTool is the path to the samba-tool binary. Variable so tests can override.
var sambaTool = "samba-tool"

// cmdTimeout is the maximum duration for any samba-tool invocation.
const cmdTimeout = 15 * time.Second

// SambaClient queries DNS via samba-tool commands.
type SambaClient struct {
	server   string // DC hostname to query (e.g., "localhost" or "bridger.example.com")
	username string // service account username
	password string // service account password
}

// NewSambaClient creates a new DNS client backed by samba-tool.
// The username is typically extracted from the bind DN (e.g., "services") and
// the password comes from the SAMBMIN_BIND_PW env var or config.
func NewSambaClient(server, username, password string) *SambaClient {
	return &SambaClient{
		server:   server,
		username: username,
		password: password,
	}
}

// Type implements Backend.
func (s *SambaClient) Type() string {
	return "samba"
}

// run executes a samba-tool command with context timeout and credentials.
func (s *SambaClient) run(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	// Append credentials
	if s.username != "" && s.password != "" {
		args = append(args, "-U", fmt.Sprintf("%s%%%s", s.username, s.password))
	}

	cmd := exec.CommandContext(ctx, sambaTool, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	slog.Debug("dns: running samba-tool", "args", args[:len(args)-1]) // don't log creds

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("samba-tool %s: %w: %s",
			strings.Join(args[:minInt(2, len(args))], " "), err, stderr.String())
	}

	return stdout.String(), nil
}

// ListZones returns all DNS zones from the DC.
func (s *SambaClient) ListZones(ctx context.Context) ([]models.DNSZone, error) {
	output, err := s.run(ctx, "dns", "zonelist", s.server)
	if err != nil {
		return nil, fmt.Errorf("list zones: %w", err)
	}

	return parseZoneList(output), nil
}

// ListRecords returns all records in a zone by querying the zone root (@).
// The recordType parameter filters results; pass "" for all types.
func (s *SambaClient) ListRecords(ctx context.Context, zone string, recordType string) ([]models.DNSRecord, error) {
	qType := "ALL"
	if recordType != "" {
		qType = recordType
	}

	output, err := s.run(ctx, "dns", "query", s.server, zone, "@", qType)
	if err != nil {
		// samba-tool returns error when no records match; treat as empty
		if strings.Contains(err.Error(), "WERR_DNS_ERROR_NAME_DOES_NOT_EXIST") {
			return []models.DNSRecord{}, nil
		}
		return nil, fmt.Errorf("list records for zone %s: %w", zone, err)
	}

	return ParseRecordOutput(output, "@"), nil
}

// GetZoneRecords queries records for a specific name within a zone.
func (s *SambaClient) GetZoneRecords(ctx context.Context, zone, name string) ([]models.DNSRecord, error) {
	output, err := s.run(ctx, "dns", "query", s.server, zone, name, "ALL")
	if err != nil {
		if strings.Contains(err.Error(), "WERR_DNS_ERROR_NAME_DOES_NOT_EXIST") {
			return []models.DNSRecord{}, nil
		}
		return nil, fmt.Errorf("get records for %s in zone %s: %w", name, zone, err)
	}

	return ParseRecordOutput(output, name), nil
}

// QueryAllRecords queries every name in a zone using wildcard '*'.
// Falls back to root '@' if wildcard is unsupported.
func (s *SambaClient) QueryAllRecords(ctx context.Context, zone string) ([]models.DNSRecord, error) {
	output, err := s.run(ctx, "dns", "query", s.server, zone, "*", "ALL")
	if err != nil {
		// Wildcard may not be supported; fall back to @ query
		slog.Debug("dns: wildcard query failed, falling back to @", "zone", zone, "error", err)
		return s.ListRecords(ctx, zone, "")
	}

	return ParseRecordOutput(output, ""), nil
}

// --- Parsing helpers ---

// parseZoneList parses the text output of `samba-tool dns zonelist`.
// Expected format:
//
//	2 zone(s) found
//	pszZoneName                 : example.com
//	Flags                       : DNS_RPC_ZONE_DSINTEGRATED DNS_RPC_ZONE_UPDATE_SECURE
//	ZoneType                    : DNS_ZONE_TYPE_PRIMARY
//	Version                     : 50
//	dwDpFlags                   : DNS_DP_AUTOCREATED DNS_DP_DOMAIN_DEFAULT DNS_DP_ENLISTED
//	pszDpFqdn                   : DomainDnsZones.example.com
//
//	pszZoneName                 : _msdcs.example.com
//	...
func parseZoneList(output string) []models.DNSZone {
	var zones []models.DNSZone
	var current *models.DNSZone

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			if current != nil {
				zones = append(zones, *current)
				current = nil
			}
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "pszZoneName":
			current = &models.DNSZone{
				Name:    val,
				Backend: "samba",
				Dynamic: true,
			}
			// Classify zone type
			if strings.Contains(val, "in-addr.arpa") || strings.Contains(val, "ip6.arpa") {
				current.Type = "reverse"
			} else {
				current.Type = "forward"
			}
		case "Flags":
			if current != nil {
				if strings.Contains(val, "DNS_RPC_ZONE_UPDATE_SECURE") {
					current.Dynamic = true
				}
			}
		}
	}
	// Flush the last zone if output doesn't end with blank line
	if current != nil {
		zones = append(zones, *current)
	}

	return zones
}

// recordLineRe matches individual record lines from samba-tool dns query output.
// Examples:
//
//	A: 10.15.15.57 (flags=600000f0, serial=110, ttl=900)
//	AAAA: fe80::1 (flags=600000f0, serial=110, ttl=3600)
//	CNAME: web.example.com. (flags=600000f0, serial=110, ttl=600)
//	MX: mail.example.com. (10) (flags=600000f0, serial=110, ttl=600)
//	SRV: dc1.example.com. 389 0 100 (flags=600000f0, serial=110, ttl=600)
//	NS: dc1.example.com. (flags=600000f0, serial=110, ttl=600)
//	TXT: "v=spf1 mx -all" (flags=600000f0, serial=110, ttl=600)
//	SOA: serial=110, refresh=900, retry=600, expire=86400, minttl=3600, ns=dc1.example.com., email=hostmaster.example.com. (flags=600000f0, serial=110, ttl=3600)
//	PTR: dc1.example.com. (flags=600000f0, serial=110, ttl=900)
var recordLineRe = regexp.MustCompile(`^\s+(A|AAAA|CNAME|MX|SRV|NS|TXT|SOA|PTR):\s+(.+)$`)

// nameLineRe matches the "Name=xxx, Records=N, Children=M" header lines.
var nameLineRe = regexp.MustCompile(`^\s*Name=([^,]*),\s*Records=(\d+),\s*Children=(\d+)`)

// ttlRe extracts TTL from the parenthesized flags section.
var ttlRe = regexp.MustCompile(`ttl=(\d+)`)

// parseRecordOutput parses the text output of `samba-tool dns query`.
// The output contains name headers followed by record detail lines.
func ParseRecordOutput(output string, defaultName string) []models.DNSRecord {
	var records []models.DNSRecord
	currentName := defaultName

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// Check for name header: "  Name=dc1, Records=2, Children=0"
		if m := nameLineRe.FindStringSubmatch(line); m != nil {
			name := m[1]
			if name == "" {
				name = "@"
			}
			currentName = name
			continue
		}

		// Check for record data line
		m := recordLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		recType := m[1]
		recData := m[2]

		rec := models.DNSRecord{
			Name:    currentName,
			Type:    recType,
			Dynamic: true, // samba DNS records are dynamic by default
		}

		// Extract TTL from the parenthesized metadata
		if ttlMatch := ttlRe.FindStringSubmatch(recData); ttlMatch != nil {
			rec.TTL, _ = strconv.Atoi(ttlMatch[1])
		}

		// Parse type-specific data
		switch recType {
		case "A", "AAAA":
			rec.Value = extractValueBeforeParens(recData)

		case "CNAME", "NS", "PTR":
			rec.Value = extractValueBeforeParens(recData)

		case "MX":
			// Format: "mail.example.com. (10) (flags=...)"
			rec.Value, rec.Priority = parseMXData(recData)

		case "SRV":
			// Format: "dc1.example.com. 389 0 100 (flags=...)"
			rec.Value, rec.Port, rec.Priority, rec.Weight = parseSRVData(recData)

		case "TXT":
			// Format: "\"v=spf1 mx -all\" (flags=...)"
			rec.Value = parseTXTData(recData)

		case "SOA":
			// Format: "serial=110, refresh=900, retry=600, expire=86400, minttl=3600, ns=dc1.example.com., email=hostmaster.example.com. (flags=...)"
			rec.Value = parseSOAData(recData)
			rec.Dynamic = false
		}

		records = append(records, rec)
	}

	return records
}

// extractValueBeforeParens returns the text before the first '(' character,
// trimmed of whitespace.
func extractValueBeforeParens(s string) string {
	if idx := strings.Index(s, "("); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return strings.TrimSpace(s)
}

// parseMXData extracts the target and priority from MX record data.
// Input: "mail.example.com. (10) (flags=600000f0, serial=110, ttl=600)"
func parseMXData(data string) (target string, priority int) {
	// Find the mail server name (before any parenthesized data)
	parts := strings.SplitN(data, "(", 2)
	target = strings.TrimSpace(parts[0])

	// Extract priority from "(10)"
	re := regexp.MustCompile(`\((\d+)\)`)
	if m := re.FindStringSubmatch(data); m != nil {
		priority, _ = strconv.Atoi(m[1])
	}

	return
}

// parseSRVData extracts target, port, priority, and weight from SRV record data.
// Input: "dc1.example.com. 389 0 100 (flags=600000f0, serial=110, ttl=600)"
func parseSRVData(data string) (target string, port, priority, weight int) {
	// Get everything before the flags paren
	clean := extractValueBeforeParens(data)
	fields := strings.Fields(clean)

	if len(fields) >= 1 {
		target = fields[0]
	}
	if len(fields) >= 2 {
		port, _ = strconv.Atoi(fields[1])
	}
	if len(fields) >= 3 {
		priority, _ = strconv.Atoi(fields[2])
	}
	if len(fields) >= 4 {
		weight, _ = strconv.Atoi(fields[3])
	}

	return
}

// parseTXTData extracts the text content from TXT record data.
// Input: "\"v=spf1 mx -all\" (flags=...)"
func parseTXTData(data string) string {
	// Strip surrounding quotes if present
	val := extractValueBeforeParens(data)
	val = strings.Trim(val, "\"")
	return val
}

// parseSOAData formats SOA record data into a readable value.
// Input: "serial=110, refresh=900, retry=600, expire=86400, minttl=3600, ns=dc1.example.com., email=hostmaster.example.com. (flags=...)"
func parseSOAData(data string) string {
	clean := extractValueBeforeParens(data)

	// Extract key fields for a human-readable SOA value
	fields := make(map[string]string)
	for _, part := range strings.Split(clean, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			fields[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	ns := fields["ns"]
	email := fields["email"]
	serial := fields["serial"]
	refresh := fields["refresh"]
	retry := fields["retry"]
	expire := fields["expire"]
	minTTL := fields["minttl"]

	return fmt.Sprintf("%s %s %s %s %s %s %s", ns, email, serial, refresh, retry, expire, minTTL)
}

// ServerInfo queries DNS server configuration via `samba-tool dns serverinfo`.
func (s *SambaClient) ServerInfo(ctx context.Context) (*models.DNSServerInfo, error) {
	output, err := s.run(ctx, "dns", "serverinfo", s.server)
	if err != nil {
		return nil, fmt.Errorf("server info: %w", err)
	}
	return ParseServerInfo(output, s.server), nil
}

// ZoneInfo queries detailed zone properties via `samba-tool dns zoneinfo`.
func (s *SambaClient) ZoneInfo(ctx context.Context, zone string) (*models.DNSZoneInfo, error) {
	output, err := s.run(ctx, "dns", "zoneinfo", s.server, zone)
	if err != nil {
		return nil, fmt.Errorf("zone info for %s: %w", zone, err)
	}
	return ParseZoneInfo(output, zone), nil
}

// QueryWithServer queries DNS records using a specific server (DC).
func (s *SambaClient) QueryWithServer(ctx context.Context, server, zone, name, recordType string) ([]models.DNSRecord, error) {
	qType := "ALL"
	if recordType != "" {
		qType = recordType
	}

	// Build args using the specified server, not the default
	args := []string{"dns", "query", server, zone, name, qType}
	if s.username != "" && s.password != "" {
		args = append(args, "-U", fmt.Sprintf("%s%%%s", s.username, s.password))
	}

	ctx2, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx2, sambaTool, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	slog.Debug("dns: query with server", "server", server, "zone", zone, "name", name, "type", qType)

	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "WERR_DNS_ERROR_NAME_DOES_NOT_EXIST") {
			return []models.DNSRecord{}, nil
		}
		return nil, fmt.Errorf("query %s on %s: %w: %s", name, server, err, stderr.String())
	}

	return ParseRecordOutput(stdout.String(), name), nil
}

// parseServerInfo parses `samba-tool dns serverinfo` output.
// Example output:
//
//	DNS information for DC 'localhost'
//	dwVersion                   : 14601
//	fBootMethod                 : DNS_BOOT_METHOD_DIRECTORY
//	fAdminConfigured            : FALSE
//	fAllowUpdate                : TRUE
//	fDsAvailable                : TRUE
//	dwForwardTimeout            : 3
//	dwRpcProtocol               : 5
//	aipForwarders               : ip4:10.15.15.1
//	...
func ParseServerInfo(output string, server string) *models.DNSServerInfo {
	info := &models.DNSServerInfo{Server: server}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "dwVersion":
			info.Version = val
		case "fAllowUpdate":
			if strings.EqualFold(val, "TRUE") {
				info.AllowUpdate = "secure"
			} else {
				info.AllowUpdate = "none"
			}
		case "aipForwarders":
			if val != "" && !strings.EqualFold(val, "NULL") && val != "[]" && val != "{}" {
				for _, f := range strings.Fields(val) {
					// Strip "ip4:" or "ip6:" prefix
					f = strings.TrimPrefix(f, "ip4:")
					f = strings.TrimPrefix(f, "ip6:")
					if f != "" && f != "[]" && f != "{}" {
						info.Forwarders = append(info.Forwarders, f)
					}
				}
			}
		}
	}

	return info
}

// parseZoneInfo parses `samba-tool dns zoneinfo` output.
// Example output:
//
//	Zone information for 'example.com'
//	pszZoneName                 : example.com
//	Flags                       : DNS_RPC_ZONE_DSINTEGRATED DNS_RPC_ZONE_UPDATE_SECURE
//	ZoneType                    : DNS_ZONE_TYPE_PRIMARY
//	fAging                      : 0
//	dwNoRefreshInterval         : 168
//	dwRefreshInterval           : 168
//	dwAvailForScavengeTime      : 0
//	aipScavengeServers          : <none>
func ParseZoneInfo(output string, zoneName string) *models.DNSZoneInfo {
	info := &models.DNSZoneInfo{
		Name:    zoneName,
		Backend: "samba",
		Status:  "healthy",
	}

	if strings.Contains(zoneName, "in-addr.arpa") || strings.Contains(zoneName, "ip6.arpa") {
		info.Type = "reverse"
	} else {
		info.Type = "forward"
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "Flags":
			if strings.Contains(val, "DNS_RPC_ZONE_UPDATE_SECURE") {
				info.DynamicUpdate = "secure"
			} else if strings.Contains(val, "DNS_RPC_ZONE_UPDATE_NONSECURE") {
				info.DynamicUpdate = "nonsecure"
			} else {
				info.DynamicUpdate = "none"
			}
		case "fAging":
			info.AgingEnabled = val != "0"
		case "dwNoRefreshInterval":
			info.NoRefreshInterval, _ = strconv.Atoi(val)
		case "dwRefreshInterval":
			info.RefreshInterval, _ = strconv.Atoi(val)
		case "aipScavengeServers":
			if val != "<none>" && val != "" {
				info.ScavengeServers = val
			}
		}
	}

	return info
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

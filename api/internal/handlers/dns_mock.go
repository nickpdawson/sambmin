package handlers

import (
	"net/http"
)

type mockDNSZone struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Backend   string `json:"backend"`
	Records   int    `json:"records"`
	Dynamic   bool   `json:"dynamic"`
	SOASerial uint32 `json:"soaSerial"`
	Status    string `json:"status"`
}

type mockDNSRecord struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"`
	Weight   int    `json:"weight,omitempty"`
	Port     int    `json:"port,omitempty"`
	Dynamic  bool   `json:"dynamic"`
}

func handleListDNSZonesMock(w http.ResponseWriter, r *http.Request) {
	zones := []mockDNSZone{
		{Name: "dzsec.net", Type: "forward", Backend: "samba", Records: 87, Dynamic: true, SOASerial: 2026021701, Status: "healthy"},
		{Name: "10.10.in-addr.arpa", Type: "reverse", Backend: "samba", Records: 34, Dynamic: true, SOASerial: 2026021501, Status: "healthy"},
		{Name: "20.10.in-addr.arpa", Type: "reverse", Backend: "bind9", Records: 12, Dynamic: true, SOASerial: 2026021401, Status: "healthy"},
		{Name: "_msdcs.dzsec.net", Type: "forward", Backend: "samba", Records: 24, Dynamic: true, SOASerial: 2026021701, Status: "healthy"},
		{Name: "lab.dzsec.net", Type: "forward", Backend: "bind9", Records: 15, Dynamic: false, SOASerial: 2026020101, Status: "warning"},
		{Name: "staging.dzsec.net", Type: "forward", Backend: "bind9", Records: 8, Dynamic: false, SOASerial: 2026011501, Status: "stale"},
	}
	respondJSON(w, http.StatusOK, map[string]any{"zones": zones, "total": len(zones)})
}

func handleListDNSRecordsMock(w http.ResponseWriter, r *http.Request) {
	zone := r.PathValue("zone")

	records := []mockDNSRecord{
		{Name: "@", Type: "SOA", Value: "dc1.dzsec.net. admin.dzsec.net. 2026021701 900 600 86400 3600", TTL: 3600, Dynamic: false},
		{Name: "@", Type: "NS", Value: "dc1.dzsec.net.", TTL: 900, Dynamic: false},
		{Name: "@", Type: "NS", Value: "dc2.dzsec.net.", TTL: 900, Dynamic: false},
		{Name: "@", Type: "A", Value: "10.10.1.10", TTL: 600, Dynamic: false},
		{Name: "dc1", Type: "A", Value: "10.10.1.10", TTL: 600, Dynamic: true},
		{Name: "dc2", Type: "A", Value: "10.10.1.11", TTL: 600, Dynamic: true},
		{Name: "dc3", Type: "A", Value: "10.20.1.10", TTL: 600, Dynamic: true},
		{Name: "mail", Type: "A", Value: "10.10.1.50", TTL: 600, Dynamic: false},
		{Name: "@", Type: "MX", Value: "mail.dzsec.net.", TTL: 600, Priority: 10, Dynamic: false},
		{Name: "www", Type: "CNAME", Value: "web.dzsec.net.", TTL: 600, Dynamic: false},
		{Name: "web", Type: "A", Value: "10.10.1.60", TTL: 600, Dynamic: false},
		{Name: "vpn", Type: "A", Value: "10.10.1.1", TTL: 300, Dynamic: false},
		{Name: "_ldap._tcp", Type: "SRV", Value: "dc1.dzsec.net.", TTL: 600, Priority: 0, Weight: 100, Port: 389, Dynamic: true},
		{Name: "_ldap._tcp", Type: "SRV", Value: "dc2.dzsec.net.", TTL: 600, Priority: 0, Weight: 100, Port: 389, Dynamic: true},
		{Name: "_kerberos._tcp", Type: "SRV", Value: "dc1.dzsec.net.", TTL: 600, Priority: 0, Weight: 100, Port: 88, Dynamic: true},
		{Name: "_kerberos._tcp", Type: "SRV", Value: "dc2.dzsec.net.", TTL: 600, Priority: 0, Weight: 100, Port: 88, Dynamic: true},
		{Name: "_gc._tcp", Type: "SRV", Value: "dc1.dzsec.net.", TTL: 600, Priority: 0, Weight: 100, Port: 3268, Dynamic: true},
		{Name: "workstation01", Type: "A", Value: "10.10.2.101", TTL: 600, Dynamic: true},
		{Name: "workstation02", Type: "A", Value: "10.10.2.102", TTL: 600, Dynamic: true},
		{Name: "laptop-jsmith", Type: "A", Value: "10.10.3.45", TTL: 600, Dynamic: true},
		{Name: "printer-floor2", Type: "A", Value: "10.10.4.10", TTL: 600, Dynamic: false},
		{Name: "@", Type: "TXT", Value: "v=spf1 mx a ip4:10.10.1.0/24 -all", TTL: 600, Dynamic: false},
		{Name: "_dmarc", Type: "TXT", Value: "v=DMARC1; p=quarantine; rua=mailto:admin@dzsec.net", TTL: 600, Dynamic: false},
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"zone":    zone,
		"records": records,
		"total":   len(records),
	})
}

func handleDNSDiagnosticsMock(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{
		"checks": []map[string]any{
			{"name": "AD SRV Records", "status": "pass", "message": "All required SRV records present (_ldap._tcp, _kerberos._tcp, _gc._tcp)"},
			{"name": "Reverse PTR Records", "status": "warning", "message": "3 A records missing reverse PTR: workstation01, workstation02, laptop-jsmith"},
			{"name": "Stale Dynamic Records", "status": "warning", "message": "2 dynamic records older than 14 days: old-laptop.dzsec.net, temp-vm.dzsec.net"},
			{"name": "SOA Serial Consistency", "status": "pass", "message": "SOA serials match across all DCs for dzsec.net"},
			{"name": "NS Record Health", "status": "pass", "message": "All NS records resolve correctly"},
			{"name": "Zone Transfer", "status": "pass", "message": "Zone transfers completing between DC1 and DC2"},
			{"name": "TTL Consistency", "status": "info", "message": "Mixed TTLs detected: 300s (vpn), 600s (most), 3600s (SOA)"},
		},
	})
}

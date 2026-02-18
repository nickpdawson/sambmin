package handlers

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	goldap "github.com/go-ldap/ldap/v3"

	"github.com/nickdawson/sambmin/internal/config"
	sambldap "github.com/nickdawson/sambmin/internal/ldap"
)

// handlerConfig holds the server configuration for handlers that need
// access to DC addresses, base DN, etc. Set by Register().
var handlerConfig *config.Config

// handleDashboardHealthLive checks each configured DC's reachability by
// performing a quick LDAP connection + RootDSE search with a 3-second timeout.
func handleDashboardHealthLive(w http.ResponseWriter, r *http.Request) {
	dcs := make([]dcStatus, 0, len(handlerConfig.DCs))
	var alerts []map[string]string

	for _, dc := range handlerConfig.DCs {
		status := probeDC(dc)
		dcs = append(dcs, status)

		if status.Status == "unreachable" {
			alerts = append(alerts, map[string]string{
				"severity": "error",
				"message":  fmt.Sprintf("Domain controller %s (%s) is unreachable.", dc.Hostname, dc.Address),
			})
		}
	}

	if alerts == nil {
		alerts = []map[string]string{}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"domainControllers": dcs,
		"alerts":            alerts,
	})
}

// probeDC performs a quick LDAP health check against a single DC.
// It connects, binds (if credentials are configured), and issues a
// RootDSE base search. The entire operation is bounded by a 3-second timeout.
func probeDC(dc config.DCConfig) dcStatus {
	port := dc.Port
	if port == 0 {
		port = 636
	}

	addr := fmt.Sprintf("%s:%d", dc.Address, port)
	useTLS := port == 636

	status := dcStatus{
		Hostname: dc.Hostname,
		Address:  dc.Address,
		Site:     dc.Site,
	}

	dialer := &net.Dialer{Timeout: 3 * time.Second}

	var conn *goldap.Conn
	var err error

	if useTLS {
		tlsCfg := &tls.Config{
			ServerName:         dc.Hostname,
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		}
		conn, err = goldap.DialURL(
			fmt.Sprintf("ldaps://%s", addr),
			goldap.DialWithTLSConfig(tlsCfg),
			goldap.DialWithDialer(dialer),
		)
	} else {
		conn, err = goldap.DialURL(
			fmt.Sprintf("ldap://%s", addr),
			goldap.DialWithDialer(dialer),
		)
	}

	if err != nil {
		slog.Warn("dashboard: DC health check failed to connect",
			"dc", dc.Hostname, "addr", addr, "error", err)
		status.Status = "unreachable"
		status.LastReplication = time.Now().Format(time.RFC3339)
		return status
	}
	defer conn.Close()

	// Bind with service account if configured
	if handlerConfig.BindDN != "" {
		if err := conn.Bind(handlerConfig.BindDN, handlerConfig.BindPW); err != nil {
			slog.Warn("dashboard: DC health check bind failed",
				"dc", dc.Hostname, "error", err)
			status.Status = "unreachable"
			status.LastReplication = time.Now().Format(time.RFC3339)
			return status
		}
	}

	// RootDSE search as health probe
	sr := goldap.NewSearchRequest(
		"",
		goldap.ScopeBaseObject,
		goldap.NeverDerefAliases,
		0, 3, false,
		"(objectClass=*)",
		[]string{"namingContexts", "defaultNamingContext"},
		nil,
	)

	_, err = conn.Search(sr)
	if err != nil {
		slog.Warn("dashboard: DC health check RootDSE search failed",
			"dc", dc.Hostname, "error", err)
		status.Status = "unreachable"
		status.LastReplication = time.Now().Format(time.RFC3339)
		return status
	}

	status.Status = "healthy"
	// Real replication metadata will come from samba-tool later;
	// for now use current time as a placeholder.
	status.LastReplication = time.Now().Format(time.RFC3339)

	slog.Debug("dashboard: DC health check passed", "dc", dc.Hostname)
	return status
}

// handleRecentActivityLive queries the directory for objects (users, groups,
// computers) that have been modified in the last 24 hours.
func handleRecentActivityLive(w http.ResponseWriter, r *http.Request) {
	if dirClient == nil {
		respondError(w, http.StatusServiceUnavailable, "directory client not available")
		return
	}

	conn, err := dirClient.Pool().Get(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError,
			"failed to get LDAP connection: "+err.Error())
		return
	}
	defer dirClient.Pool().Put(conn)

	// Build LDAP generalized time for 24 hours ago
	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	ldapTimestamp := cutoff.Format("20060102150405.0Z")

	filter := fmt.Sprintf(
		"(&(|(objectClass=user)(objectClass=group)(objectClass=computer))(whenChanged>=%s))",
		ldapTimestamp,
	)

	sr := goldap.NewSearchRequest(
		handlerConfig.BaseDN,
		goldap.ScopeWholeSubtree,
		goldap.NeverDerefAliases,
		100, // size limit — cap at 100 recent changes
		5,   // time limit seconds
		false,
		filter,
		[]string{
			sambldap.AttrDN,
			sambldap.AttrCN,
			sambldap.AttrObjectClass,
			sambldap.AttrWhenChanged,
		},
		nil,
	)

	result, err := conn.Search(sr)
	if err != nil {
		slog.Warn("dashboard: recent activity LDAP search failed", "error", err)
		respondError(w, http.StatusInternalServerError,
			"failed to query recent activity: "+err.Error())
		return
	}

	activities := make([]recentActivity, 0, len(result.Entries))
	for _, entry := range result.Entries {
		whenChanged := entry.GetAttributeValue(sambldap.AttrWhenChanged)
		ts := parseActivityTimestamp(whenChanged)

		activities = append(activities, recentActivity{
			Timestamp: ts.Format(time.RFC3339),
			Actor:     "system",
			Action:    "Modified",
			Object:    entry.DN,
			Success:   true,
		})
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"activities": activities,
	})
}

// parseActivityTimestamp parses AD generalized time formats.
func parseActivityTimestamp(s string) time.Time {
	if s == "" {
		return time.Now()
	}
	for _, layout := range []string{
		"20060102150405.0Z",
		"20060102150405Z",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Now()
}

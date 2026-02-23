package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/nickdawson/sambmin/internal/validate"
)

// --- DNS Zone CRUD ---

type createDNSZoneRequest struct {
	Name string `json:"name"`
}

func handleCreateDNSZone(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req createDNSZoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "zone name required")
		return
	}
	if err := validate.DNSName(req.Name); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	server := "localhost"
	if _, err := runSambaTool(r.Context(), sess, "dns", "zonecreate", server, req.Name); err != nil {
		slog.Error("DNS zone create failed", "zone", req.Name, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, "DNS zone creation failed")
		return
	}

	slog.Info("DNS zone created", "zone", req.Name, "actor", sess.Username)
	respondJSON(w, http.StatusCreated, map[string]any{"success": true, "zone": req.Name})
}

func handleDeleteDNSZone(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	zone := r.PathValue("zone")
	if zone == "" {
		respondError(w, http.StatusBadRequest, "zone name required")
		return
	}
	if err := validate.DNSName(zone); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	server := "localhost"
	if _, err := runSambaTool(r.Context(), sess, "dns", "zonedelete", server, zone); err != nil {
		slog.Error("DNS zone delete failed", "zone", zone, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, "DNS zone deletion failed")
		return
	}

	slog.Info("DNS zone deleted", "zone", zone, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "zone": zone})
}

// --- DNS Record CRUD ---

type dnsRecordRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
	Weight   int    `json:"weight"`
	Port     int    `json:"port"`
}

func handleCreateDNSRecord(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	zone := r.PathValue("zone")
	if zone == "" {
		respondError(w, http.StatusBadRequest, "zone name required")
		return
	}

	var req dnsRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Type == "" || req.Value == "" {
		respondError(w, http.StatusBadRequest, "name, type, and value required")
		return
	}
	if err := validate.DNSName(req.Name); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validate.DNSType(req.Type); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	server := "localhost"
	args := []string{"dns", "add", server, zone, req.Name, req.Type, req.Value}

	if _, err := runSambaTool(r.Context(), sess, args...); err != nil {
		slog.Error("DNS record create failed", "zone", zone, "name", req.Name, "type", req.Type, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, "DNS record creation failed")
		return
	}

	slog.Info("DNS record created", "zone", zone, "name", req.Name, "type", req.Type, "actor", sess.Username)
	respondJSON(w, http.StatusCreated, map[string]any{"success": true})
}

type updateDNSRecordRequest struct {
	Type     string `json:"type"`
	OldValue string `json:"oldValue"`
	NewValue string `json:"newValue"`
}

func handleUpdateDNSRecord(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	zone := r.PathValue("zone")
	name := r.PathValue("name")
	if zone == "" || name == "" {
		respondError(w, http.StatusBadRequest, "zone and record name required")
		return
	}

	var req updateDNSRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Type == "" || req.OldValue == "" || req.NewValue == "" {
		respondError(w, http.StatusBadRequest, "type, oldValue, and newValue required")
		return
	}
	if err := validate.DNSType(req.Type); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	server := "localhost"
	args := []string{"dns", "update", server, zone, name, req.Type, req.OldValue, req.NewValue}

	if _, err := runSambaTool(r.Context(), sess, args...); err != nil {
		slog.Error("DNS record update failed", "zone", zone, "name", name, "type", req.Type, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, "DNS record update failed")
		return
	}

	slog.Info("DNS record updated", "zone", zone, "name", name, "type", req.Type, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

func handleDeleteDNSRecord(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	zone := r.PathValue("zone")
	name := r.PathValue("name")
	if zone == "" || name == "" {
		respondError(w, http.StatusBadRequest, "zone and record name required")
		return
	}

	// For delete, we need the record type and value from query params
	recType := r.URL.Query().Get("type")
	value := r.URL.Query().Get("value")
	if recType == "" || value == "" {
		respondError(w, http.StatusBadRequest, "type and value query params required")
		return
	}
	if err := validate.DNSType(recType); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	server := "localhost"
	args := []string{"dns", "delete", server, zone, name, recType, value}

	if _, err := runSambaTool(r.Context(), sess, args...); err != nil {
		slog.Error("DNS record delete failed", "zone", zone, "name", name, "type", recType, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, "DNS record deletion failed")
		return
	}

	slog.Info("DNS record deleted", "zone", zone, "name", name, "type", recType, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

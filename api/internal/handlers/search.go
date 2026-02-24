package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nickdawson/sambmin/internal/auth"
	"github.com/nickdawson/sambmin/internal/models"
	"github.com/nickdawson/sambmin/internal/validate"

	goldap "github.com/go-ldap/ldap/v3"
)

// savedQueries is an in-memory store for saved queries.
var (
	savedQueries   = make(map[string]models.SavedQuery)
	savedQueriesMu sync.RWMutex
	queryIDCounter int
)

// handleSearch executes an LDAP search using visual filters or a raw filter.
func handleSearch(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	if dirClient == nil {
		respondError(w, http.StatusServiceUnavailable, "directory not available")
		return
	}

	var req models.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Build the LDAP filter
	var filter string
	if req.RawFilter != "" {
		// Validate raw filter
		if err := validate.RawFilter(req.RawFilter); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		// Raw filters require admin role
		if !auth.HasRole(sess, auth.RoleAdmin) {
			respondError(w, http.StatusForbidden, "raw LDAP filters require admin privileges")
			return
		}
		filter = req.RawFilter
	} else if len(req.Filters) > 0 {
		filter = buildFilterFromVisual(req.ObjectType, req.Filters)
	} else {
		respondError(w, http.StatusBadRequest, "either rawFilter or filters required")
		return
	}

	// Determine base DN
	baseDN := req.BaseDN
	if baseDN == "" && handlerConfig != nil {
		baseDN = handlerConfig.BaseDN
	}
	if baseDN == "" {
		respondError(w, http.StatusBadRequest, "baseDN required")
		return
	}
	// Validate baseDN ends with configured domain suffix
	if handlerConfig != nil && handlerConfig.BaseDN != "" {
		if err := validate.BaseDN(baseDN, handlerConfig.BaseDN); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	// Determine scope
	scope := goldap.ScopeWholeSubtree
	switch req.Scope {
	case "base":
		scope = goldap.ScopeBaseObject
	case "one":
		scope = goldap.ScopeSingleLevel
	}

	// Determine attributes to return
	attrs := req.Attributes
	if len(attrs) == 0 {
		attrs = []string{"dn", "objectClass", "cn", "sAMAccountName", "displayName", "mail", "description", "whenCreated"}
	}
	// Block sensitive attributes
	if err := validate.Attributes(attrs); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	results, err := dirClient.Search(r.Context(), baseDN, scope, filter, attrs)
	if err != nil {
		slog.Error("LDAP search failed", "filter", filter, "actor", sess.Username, "error", err)
		respondError(w, http.StatusInternalServerError, "search failed")
		return
	}

	slog.Info("search executed", "filter", filter, "results", len(results), "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{
		"results": results,
		"count":   len(results),
		"filter":  filter,
	})
}

// handleListSavedQueries returns all saved queries.
func handleListSavedQueries(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	savedQueriesMu.RLock()
	queries := make([]models.SavedQuery, 0, len(savedQueries))
	for _, q := range savedQueries {
		queries = append(queries, q)
	}
	savedQueriesMu.RUnlock()

	respondJSON(w, http.StatusOK, queries)
}

// handleCreateSavedQuery saves a new query.
func handleCreateSavedQuery(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	var req struct {
		Name        string              `json:"name"`
		Description string              `json:"description"`
		Request     models.SearchRequest `json:"request"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "query name required")
		return
	}

	savedQueriesMu.Lock()
	queryIDCounter++
	id := fmt.Sprintf("q%d", queryIDCounter)
	query := models.SavedQuery{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Request:     req.Request,
		CreatedBy:   sess.Username,
		CreatedAt:   time.Now(),
	}
	savedQueries[id] = query
	savedQueriesMu.Unlock()

	slog.Info("query saved", "id", id, "name", req.Name, "actor", sess.Username)
	respondJSON(w, http.StatusCreated, query)
}

// handleDeleteSavedQuery deletes a saved query by ID.
func handleDeleteSavedQuery(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "query ID required")
		return
	}

	savedQueriesMu.Lock()
	if _, ok := savedQueries[id]; !ok {
		savedQueriesMu.Unlock()
		respondError(w, http.StatusNotFound, "query not found")
		return
	}
	delete(savedQueries, id)
	savedQueriesMu.Unlock()

	slog.Info("query deleted", "id", id, "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

// buildFilterFromVisual constructs an LDAP filter from visual filter conditions.
func buildFilterFromVisual(objectType string, filters []models.SearchFilter) string {
	var parts []string

	// Add object class filter based on type
	switch objectType {
	case "user":
		parts = append(parts, "(&(objectClass=user)(!(objectClass=computer)))")
	case "group":
		parts = append(parts, "(objectClass=group)")
	case "computer":
		parts = append(parts, "(objectClass=computer)")
	case "contact":
		parts = append(parts, "(&(objectClass=contact)(!(objectClass=user)))")
	default:
		// "all" or empty — no object type filter
	}

	// Build attribute conditions
	for _, f := range filters {
		escaped := goldap.EscapeFilter(f.Value)
		attr := f.Attribute

		var condition string
		switch f.Operator {
		case "equals":
			condition = fmt.Sprintf("(%s=%s)", attr, escaped)
		case "contains":
			condition = fmt.Sprintf("(%s=*%s*)", attr, escaped)
		case "startsWith":
			condition = fmt.Sprintf("(%s=%s*)", attr, escaped)
		case "endsWith":
			condition = fmt.Sprintf("(%s=*%s)", attr, escaped)
		case "present":
			condition = fmt.Sprintf("(%s=*)", attr)
		case "notPresent":
			condition = fmt.Sprintf("(!(%s=*))", attr)
		case "greaterThan":
			condition = fmt.Sprintf("(%s>=%s)", attr, escaped)
		case "lessThan":
			condition = fmt.Sprintf("(%s<=%s)", attr, escaped)
		case "bitwiseAnd":
			condition = fmt.Sprintf("(%s:1.2.840.113556.1.4.803:=%s)", attr, escaped)
		case "bitwiseOr":
			condition = fmt.Sprintf("(%s:1.2.840.113556.1.4.804:=%s)", attr, escaped)
		default:
			condition = fmt.Sprintf("(%s=%s)", attr, escaped)
		}
		parts = append(parts, condition)
	}

	if len(parts) == 0 {
		return "(objectClass=*)"
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return fmt.Sprintf("(&%s)", strings.Join(parts, ""))
}

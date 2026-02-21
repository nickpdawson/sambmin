package handlers

import (
	"net/http"

	"github.com/nickdawson/sambmin/internal/directory"
)

// --- Groups ---

func handleListGroupsLive(w http.ResponseWriter, r *http.Request) {
	opts := directory.ListGroupsOptions{
		Search:       r.URL.Query().Get("q"),
		SecurityOnly: r.URL.Query().Get("type") == "security",
	}

	groups, err := dirClient.ListGroups(r.Context(), opts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list groups: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"groups": groups,
		"total":  len(groups),
	})
}

func handleGetGroupLive(w http.ResponseWriter, r *http.Request) {
	dn := r.PathValue("dn")
	group, err := dirClient.GetGroup(r.Context(), dn)
	if err != nil {
		respondError(w, http.StatusNotFound, "group not found: "+err.Error())
		return
	}
	respondJSON(w, http.StatusOK, group)
}

// --- Computers ---

func handleListComputersLive(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("q")
	computers, err := dirClient.ListComputers(r.Context(), search)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list computers: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"computers": computers,
		"total":     len(computers),
	})
}

func handleGetComputerLive(w http.ResponseWriter, r *http.Request) {
	dn := r.PathValue("dn")
	comp, err := dirClient.GetComputer(r.Context(), dn)
	if err != nil {
		respondError(w, http.StatusNotFound, "computer not found: "+err.Error())
		return
	}
	respondJSON(w, http.StatusOK, comp)
}

// --- OUs ---

func handleListOUsLive(w http.ResponseWriter, r *http.Request) {
	ous, err := dirClient.ListOUs(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list OUs: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"ous":   ous,
		"total": len(ous),
	})
}

func handleOUTreeLive(w http.ResponseWriter, r *http.Request) {
	tree, err := dirClient.GetOUTree(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get OU tree: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"tree": tree,
	})
}

// handleOUTreeFullLive returns the complete OU tree with all child objects.
// GET /api/ous/tree/full
func handleOUTreeFullLive(w http.ResponseWriter, r *http.Request) {
	tree, contents, err := dirClient.GetFullTree(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get full tree: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"tree":     tree,
		"contents": contents,
	})
}

// handleOUContentsLive returns the direct child objects of an OU.
// GET /api/ous/{dn}/contents
func handleOUContentsLive(w http.ResponseWriter, r *http.Request) {
	dn := r.PathValue("dn")
	if dn == "" {
		respondError(w, http.StatusBadRequest, "OU DN required")
		return
	}

	children, err := dirClient.ListOUContents(r.Context(), dn)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list OU contents: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"children": children,
		"total":    len(children),
		"dn":       dn,
	})
}

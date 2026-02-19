package handlers

import (
	"net/http"
)

func handleListContactsLive(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("q")
	contacts, err := dirClient.ListContacts(r.Context(), search)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list contacts: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"contacts": contacts,
		"total":    len(contacts),
	})
}

func handleGetContactLive(w http.ResponseWriter, r *http.Request) {
	dn := r.PathValue("dn")
	contact, err := dirClient.GetContact(r.Context(), dn)
	if err != nil {
		respondError(w, http.StatusNotFound, "contact not found: "+err.Error())
		return
	}
	respondJSON(w, http.StatusOK, contact)
}

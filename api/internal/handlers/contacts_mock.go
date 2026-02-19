package handlers

import (
	"net/http"
)

func handleListContactsMock(w http.ResponseWriter, r *http.Request) {
	contacts := []map[string]any{
		{
			"dn": "CN=External Vendor,OU=Contacts,DC=dzsec,DC=net", "name": "External Vendor",
			"displayName": "External Vendor", "givenName": "External", "sn": "Vendor",
			"mail": "vendor@example.com", "description": "Third-party vendor contact",
			"department": "Procurement", "title": "Account Manager", "company": "Acme Corp",
			"phone": "+1-555-0100", "mobile": "", "office": "", "streetAddress": "",
			"city": "", "state": "", "postalCode": "", "country": "",
			"whenCreated": "2024-06-01T10:00:00Z", "whenChanged": "2024-06-01T10:00:00Z",
			"memberOf": []string{},
		},
		{
			"dn": "CN=Partner Rep,OU=Contacts,DC=dzsec,DC=net", "name": "Partner Rep",
			"displayName": "Partner Rep", "givenName": "Partner", "sn": "Rep",
			"mail": "partner@partner.org", "description": "Strategic partner liaison",
			"department": "", "title": "Director", "company": "Partner Org",
			"phone": "+1-555-0200", "mobile": "+1-555-0201", "office": "Building A",
			"streetAddress": "123 Main St", "city": "Bozeman", "state": "MT",
			"postalCode": "59715", "country": "US",
			"whenCreated": "2024-03-15T09:00:00Z", "whenChanged": "2025-01-10T14:00:00Z",
			"memberOf": []string{"CN=VPN-External,OU=Groups,DC=dzsec,DC=net"},
		},
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"contacts": contacts,
		"total":    len(contacts),
	})
}

func handleGetContactMock(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{
		"dn": "CN=External Vendor,OU=Contacts,DC=dzsec,DC=net", "name": "External Vendor",
		"displayName": "External Vendor", "givenName": "External", "sn": "Vendor",
		"mail": "vendor@example.com", "description": "Third-party vendor contact",
		"department": "Procurement", "title": "Account Manager", "company": "Acme Corp",
		"phone": "+1-555-0100", "mobile": "", "office": "", "streetAddress": "",
		"city": "", "state": "", "postalCode": "", "country": "",
		"whenCreated": "2024-06-01T10:00:00Z", "whenChanged": "2024-06-01T10:00:00Z",
		"memberOf": []string{},
	})
}

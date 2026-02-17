package handlers

import (
	"net/http"
	"time"
)

// Mock user data for local development

type mockUser struct {
	DN              string   `json:"dn"`
	SamAccountName  string   `json:"samAccountName"`
	DisplayName     string   `json:"displayName"`
	GivenName       string   `json:"givenName"`
	Surname         string   `json:"sn"`
	Email           string   `json:"mail"`
	UPN             string   `json:"userPrincipalName"`
	Department      string   `json:"department"`
	Title           string   `json:"title"`
	Enabled         bool     `json:"enabled"`
	LockedOut       bool     `json:"lockedOut"`
	LastLogon       string   `json:"lastLogon"`
	WhenCreated     string   `json:"whenCreated"`
	MemberOf        []string `json:"memberOf"`
}

func handleListUsersMock(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	users := []mockUser{
		{
			DN: "CN=Nick Dawson,OU=Admins,DC=dzsec,DC=net", SamAccountName: "ndawson",
			DisplayName: "Nick Dawson", GivenName: "Nick", Surname: "Dawson",
			Email: "nick@dzsec.net", UPN: "ndawson@dzsec.net",
			Department: "IT", Title: "Administrator",
			Enabled: true, LockedOut: false,
			LastLogon: now.Add(-10 * time.Minute).Format(time.RFC3339),
			WhenCreated: "2024-01-15T09:00:00Z",
			MemberOf: []string{"Domain Admins", "Schema Admins", "Enterprise Admins"},
		},
		{
			DN: "CN=John Smith,OU=Users,DC=dzsec,DC=net", SamAccountName: "jsmith",
			DisplayName: "John Smith", GivenName: "John", Surname: "Smith",
			Email: "jsmith@dzsec.net", UPN: "jsmith@dzsec.net",
			Department: "Engineering", Title: "Senior Engineer",
			Enabled: true, LockedOut: false,
			LastLogon: now.Add(-1 * time.Hour).Format(time.RFC3339),
			WhenCreated: "2024-03-20T14:30:00Z",
			MemberOf: []string{"Domain Users", "VPN-Users", "Engineering"},
		},
		{
			DN: "CN=Mary Jones,OU=Users,DC=dzsec,DC=net", SamAccountName: "mjones",
			DisplayName: "Mary Jones", GivenName: "Mary", Surname: "Jones",
			Email: "mjones@dzsec.net", UPN: "mjones@dzsec.net",
			Department: "Marketing", Title: "Marketing Manager",
			Enabled: true, LockedOut: true,
			LastLogon: now.Add(-24 * time.Hour).Format(time.RFC3339),
			WhenCreated: "2024-06-01T10:00:00Z",
			MemberOf: []string{"Domain Users", "Marketing"},
		},
		{
			DN: "CN=Bob Wilson,OU=Users,DC=dzsec,DC=net", SamAccountName: "bwilson",
			DisplayName: "Bob Wilson", GivenName: "Bob", Surname: "Wilson",
			Email: "bwilson@dzsec.net", UPN: "bwilson@dzsec.net",
			Department: "Engineering", Title: "DevOps Engineer",
			Enabled: true, LockedOut: true,
			LastLogon: now.Add(-2 * time.Hour).Format(time.RFC3339),
			WhenCreated: "2024-04-15T08:00:00Z",
			MemberOf: []string{"Domain Users", "VPN-Users", "Engineering", "IT-Ops"},
		},
		{
			DN: "CN=Sarah Chen,OU=Users,DC=dzsec,DC=net", SamAccountName: "schen",
			DisplayName: "Sarah Chen", GivenName: "Sarah", Surname: "Chen",
			Email: "schen@dzsec.net", UPN: "schen@dzsec.net",
			Department: "Finance", Title: "Controller",
			Enabled: true, LockedOut: false,
			LastLogon: now.Add(-30 * time.Minute).Format(time.RFC3339),
			WhenCreated: "2024-02-01T09:00:00Z",
			MemberOf: []string{"Domain Users", "Finance", "Budget-Approvers"},
		},
		{
			DN: "CN=Alex Contractor,OU=Contractors,DC=dzsec,DC=net", SamAccountName: "contractor01",
			DisplayName: "Alex Contractor", GivenName: "Alex", Surname: "Contractor",
			Email: "alex@contractor.com", UPN: "contractor01@dzsec.net",
			Department: "External", Title: "Contractor",
			Enabled: false, LockedOut: false,
			LastLogon: now.Add(-72 * time.Hour).Format(time.RFC3339),
			WhenCreated: "2025-01-10T09:00:00Z",
			MemberOf: []string{"Domain Users"},
		},
		{
			DN: "CN=Tom Davis,OU=Users,DC=dzsec,DC=net", SamAccountName: "tdavis",
			DisplayName: "Tom Davis", GivenName: "Tom", Surname: "Davis",
			Email: "tdavis@dzsec.net", UPN: "tdavis@dzsec.net",
			Department: "Engineering", Title: "Software Engineer",
			Enabled: true, LockedOut: true,
			LastLogon: now.Add(-4 * time.Hour).Format(time.RFC3339),
			WhenCreated: "2024-08-01T09:00:00Z",
			MemberOf: []string{"Domain Users", "Engineering", "VPN-Users"},
		},
		{
			DN: "CN=Lisa Park,OU=Users,DC=dzsec,DC=net", SamAccountName: "lpark",
			DisplayName: "Lisa Park", GivenName: "Lisa", Surname: "Park",
			Email: "lpark@dzsec.net", UPN: "lpark@dzsec.net",
			Department: "HR", Title: "HR Director",
			Enabled: true, LockedOut: false,
			LastLogon: now.Add(-15 * time.Minute).Format(time.RFC3339),
			WhenCreated: "2024-01-20T09:00:00Z",
			MemberOf: []string{"Domain Users", "HR", "Management"},
		},
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"users": users,
		"total": len(users),
	})
}

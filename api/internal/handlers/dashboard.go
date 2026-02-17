package handlers

import (
	"net/http"
	"time"
)

// Mock dashboard data for local development without a DC connection

type dcStatus struct {
	Hostname       string `json:"hostname"`
	Address        string `json:"address"`
	Site           string `json:"site"`
	Status         string `json:"status"`
	LastReplication string `json:"lastReplication"`
	FSMORoles      []string `json:"fsmoRoles"`
	IsGC           bool   `json:"isGlobalCatalog"`
}

type dashboardMetrics struct {
	TotalUsers     int `json:"totalUsers"`
	TotalComputers int `json:"totalComputers"`
	TotalGroups    int `json:"totalGroups"`
	TotalDNSZones  int `json:"totalDNSZones"`
	LockedAccounts int `json:"lockedAccounts"`
	DisabledUsers  int `json:"disabledUsers"`
}

type recentActivity struct {
	Timestamp string `json:"timestamp"`
	Actor     string `json:"actor"`
	Action    string `json:"action"`
	Object    string `json:"object"`
	Success   bool   `json:"success"`
}

func handleDashboardHealthMock(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	data := map[string]any{
		"domainControllers": []dcStatus{
			{
				Hostname:       "DC1-BOZEMAN",
				Address:        "10.10.1.10",
				Site:           "Bozeman",
				Status:         "healthy",
				LastReplication: now.Add(-2 * time.Minute).Format(time.RFC3339),
				FSMORoles:      []string{"PDC Emulator", "RID Master", "Infrastructure Master"},
				IsGC:           true,
			},
			{
				Hostname:       "DC2-BRIDGER",
				Address:        "10.10.1.11",
				Site:           "Bozeman",
				Status:         "healthy",
				LastReplication: now.Add(-5 * time.Minute).Format(time.RFC3339),
				FSMORoles:      []string{"Schema Master", "Domain Naming Master"},
				IsGC:           true,
			},
			{
				Hostname:       "DC3-SEATTLE",
				Address:        "10.20.1.10",
				Site:           "Seattle",
				Status:         "warning",
				LastReplication: now.Add(-47 * time.Minute).Format(time.RFC3339),
				FSMORoles:      nil,
				IsGC:           false,
			},
		},
		"alerts": []map[string]string{
			{
				"severity": "warning",
				"message":  "Replication from DC3-SEATTLE is 47 minutes behind. Expected: < 15 minutes.",
			},
			{
				"severity": "info",
				"message":  "3 accounts locked out in the past hour.",
			},
		},
	}
	respondJSON(w, http.StatusOK, data)
}

func handleDashboardMetricsMock(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, dashboardMetrics{
		TotalUsers:     247,
		TotalComputers: 89,
		TotalGroups:    34,
		TotalDNSZones:  6,
		LockedAccounts: 3,
		DisabledUsers:  12,
	})
}

func handleRecentActivityMock(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	respondJSON(w, http.StatusOK, map[string]any{
		"activities": []recentActivity{
			{Timestamp: now.Add(-3 * time.Minute).Format(time.RFC3339), Actor: "administrator", Action: "Password Reset", Object: "CN=jsmith,OU=Users,DC=dzsec,DC=net", Success: true},
			{Timestamp: now.Add(-12 * time.Minute).Format(time.RFC3339), Actor: "administrator", Action: "User Disabled", Object: "CN=contractor01,OU=Contractors,DC=dzsec,DC=net", Success: true},
			{Timestamp: now.Add(-25 * time.Minute).Format(time.RFC3339), Actor: "helpdesk", Action: "Account Unlock", Object: "CN=mjones,OU=Users,DC=dzsec,DC=net", Success: true},
			{Timestamp: now.Add(-1 * time.Hour).Format(time.RFC3339), Actor: "administrator", Action: "DNS Record Created", Object: "mail.dzsec.net (A: 10.10.1.50)", Success: true},
			{Timestamp: now.Add(-2 * time.Hour).Format(time.RFC3339), Actor: "administrator", Action: "Group Member Added", Object: "CN=VPN-Users,OU=Groups,DC=dzsec,DC=net", Success: true},
			{Timestamp: now.Add(-3 * time.Hour).Format(time.RFC3339), Actor: "administrator", Action: "User Created", Object: "CN=newuser,OU=Users,DC=dzsec,DC=net", Success: true},
			{Timestamp: now.Add(-5 * time.Hour).Format(time.RFC3339), Actor: "administrator", Action: "GPO Linked", Object: "Default Domain Policy → OU=Workstations", Success: true},
			{Timestamp: now.Add(-8 * time.Hour).Format(time.RFC3339), Actor: "administrator", Action: "Force Replication", Object: "DC1-BOZEMAN → DC3-SEATTLE", Success: false},
		},
	})
}

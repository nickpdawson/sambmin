package models

import "time"

// User represents an AD user account
type User struct {
	DN              string    `json:"dn"`
	SamAccountName  string    `json:"samAccountName"`
	DisplayName     string    `json:"displayName"`
	GivenName       string    `json:"givenName"`
	Surname         string    `json:"sn"`
	Email           string    `json:"mail"`
	UPN             string    `json:"userPrincipalName"`
	Description     string    `json:"description"`
	Department      string    `json:"department"`
	Title           string    `json:"title"`
	Company         string    `json:"company"`
	Manager         string    `json:"manager"`
	Office          string    `json:"office"`
	Street          string    `json:"streetAddress"`
	City            string    `json:"city"`
	State           string    `json:"state"`
	PostalCode      string    `json:"postalCode"`
	Country         string    `json:"country"`
	Phone           string    `json:"phone"`
	Mobile          string    `json:"mobile"`
	Enabled         bool      `json:"enabled"`
	LockedOut       bool      `json:"lockedOut"`
	PasswordExpired bool      `json:"passwordExpired"`
	AccountExpires  time.Time `json:"accountExpires"`
	PwdLastSet      time.Time `json:"pwdLastSet"`
	BadPwdCount     int       `json:"badPwdCount"`
	LastLogon       time.Time `json:"lastLogon"`
	WhenCreated     time.Time `json:"whenCreated"`
	WhenChanged     time.Time `json:"whenChanged"`
	MemberOf        []string  `json:"memberOf"`
}

// Group represents an AD security or distribution group
type Group struct {
	DN             string   `json:"dn"`
	Name           string   `json:"name"`
	SamAccountName string   `json:"samAccountName"`
	Description    string   `json:"description"`
	GroupType      string   `json:"groupType"` // "security" or "distribution"
	GroupScope     string   `json:"groupScope"` // "global", "domainLocal", "universal"
	Members        []string `json:"members"`
	MemberOf       []string `json:"memberOf"`
}

// Computer represents an AD machine account
type Computer struct {
	DN             string    `json:"dn"`
	Name           string    `json:"name"`
	SamAccountName string    `json:"samAccountName"`
	DNSHostName    string    `json:"dnsHostName"`
	OS             string    `json:"operatingSystem"`
	OSVersion      string    `json:"operatingSystemVersion"`
	Site           string    `json:"site"`
	Enabled        bool      `json:"enabled"`
	LastLogon      time.Time `json:"lastLogon"`
	WhenCreated    time.Time `json:"whenCreated"`
}

// OU represents an Organizational Unit
type OU struct {
	DN          string `json:"dn"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ChildCount  int    `json:"childCount"`
}

// DNSZone represents a DNS zone
type DNSZone struct {
	Name       string `json:"name"`
	Type       string `json:"type"` // "forward" or "reverse"
	Backend    string `json:"backend"` // "samba" or "bind9"
	Records    int    `json:"records"`
	Dynamic    bool   `json:"dynamic"`
	SOASerial  uint32 `json:"soaSerial"`
}

// DNSRecord represents a DNS resource record
type DNSRecord struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // A, AAAA, CNAME, MX, SRV, TXT, NS, PTR, SOA
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"` // MX, SRV
	Weight   int    `json:"weight,omitempty"`   // SRV
	Port     int    `json:"port,omitempty"`     // SRV
	Dynamic  bool   `json:"dynamic"`
}

// DCStatus represents a domain controller's health
type DCStatus struct {
	Hostname       string    `json:"hostname"`
	Address        string    `json:"address"`
	Site           string    `json:"site"`
	Status         string    `json:"status"` // "healthy", "warning", "error", "unreachable"
	LastReplication time.Time `json:"lastReplication"`
	FSMORoles      []string  `json:"fsmoRoles"`
	IsGC           bool      `json:"isGlobalCatalog"`
}

// ReplicationLink represents a replication partnership between DCs
type ReplicationLink struct {
	SourceDC       string    `json:"sourceDC"`
	DestDC         string    `json:"destDC"`
	NamingContext  string    `json:"namingContext"`
	LastSync       time.Time `json:"lastSync"`
	Status         string    `json:"status"` // "current", "behind", "failed"
	PendingChanges int       `json:"pendingChanges"`
}

// Site represents an AD site
type Site struct {
	Name    string   `json:"name"`
	Subnets []string `json:"subnets"`
	DCs     []string `json:"dcs"`
}

// AuditEntry represents an action logged for audit
type AuditEntry struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	ObjectDN  string    `json:"objectDN"`
	ObjectType string   `json:"objectType"`
	DC        string    `json:"dc"`
	Success   bool      `json:"success"`
	Details   string    `json:"details"`
	SourceIP  string    `json:"sourceIP"`
}

// DashboardMetrics for the overview dashboard
type DashboardMetrics struct {
	TotalUsers     int `json:"totalUsers"`
	TotalComputers int `json:"totalComputers"`
	TotalGroups    int `json:"totalGroups"`
	TotalDNSZones  int `json:"totalDNSZones"`
	LockedAccounts int `json:"lockedAccounts"`
	DisabledUsers  int `json:"disabledUsers"`
}

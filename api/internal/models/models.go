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

// Contact represents an AD contact object (non-security principal)
type Contact struct {
	DN          string `json:"dn"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	GivenName   string `json:"givenName"`
	Surname     string `json:"sn"`
	Email       string `json:"mail"`
	Description string `json:"description"`
	Department  string `json:"department"`
	Title       string `json:"title"`
	Company     string `json:"company"`
	Office      string `json:"office"`
	Phone       string `json:"phone"`
	Mobile      string `json:"mobile"`
	Street      string `json:"streetAddress"`
	City        string `json:"city"`
	State       string `json:"state"`
	PostalCode  string `json:"postalCode"`
	Country     string `json:"country"`
	WhenCreated time.Time `json:"whenCreated"`
	WhenChanged time.Time `json:"whenChanged"`
	MemberOf    []string  `json:"memberOf"`
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

// DNSServerInfo represents DNS server configuration from samba-tool dns serverinfo
type DNSServerInfo struct {
	Server       string   `json:"server"`
	Forwarders   []string `json:"forwarders"`
	RootHints    bool     `json:"rootHints"`
	AllowUpdate  string   `json:"allowUpdate"`
	Zones        int      `json:"zones"`
	Version      string   `json:"version"`
}

// DNSZoneInfo represents detailed zone properties including aging/scavenging
type DNSZoneInfo struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	Backend           string `json:"backend"`
	DynamicUpdate     string `json:"dynamicUpdate"`    // "none", "secure", "nonsecure"
	AgingEnabled      bool   `json:"agingEnabled"`
	NoRefreshInterval int    `json:"noRefreshInterval"` // hours
	RefreshInterval   int    `json:"refreshInterval"`   // hours
	ScavengeServers   string `json:"scavengeServers"`
	Records           int    `json:"records"`
	SOASerial         uint32 `json:"soaSerial"`
	Status            string `json:"status"` // "healthy", "warning", "stale"
}

// DNSQueryRequest represents a DNS query request targeting a specific DC
type DNSQueryRequest struct {
	Server string `json:"server"` // DC to query
	Zone   string `json:"zone"`
	Name   string `json:"name"`
	Type   string `json:"type"` // A, AAAA, SRV, etc. or ALL
}

// DNSQueryResult represents the result of a DNS query
type DNSQueryResult struct {
	Server  string      `json:"server"`
	Zone    string      `json:"zone"`
	Name    string      `json:"name"`
	Records []DNSRecord `json:"records"`
	Error   string      `json:"error,omitempty"`
}

// SRVValidationEntry represents one SRV record check for a specific DC
type SRVValidationEntry struct {
	Record  string `json:"record"`  // e.g., "_ldap._tcp"
	DC      string `json:"dc"`
	Status  string `json:"status"`  // "pass", "fail", "error"
	Targets int    `json:"targets"` // number of records found
	Message string `json:"message,omitempty"`
}

// PasswordPolicy represents the domain-wide password policy settings
type PasswordPolicy struct {
	MinLength           int    `json:"minLength"`
	MaxAge              string `json:"maxAge"`              // e.g., "42 days"
	MinAge              string `json:"minAge"`              // e.g., "1 day"
	HistoryLength       int    `json:"historyLength"`
	Complexity          bool   `json:"complexity"`
	ReversibleEncryption bool  `json:"reversibleEncryption"`
	LockoutThreshold    int    `json:"lockoutThreshold"`
	LockoutDuration     string `json:"lockoutDuration"`     // e.g., "30 minutes"
	LockoutWindow       string `json:"lockoutWindow"`       // e.g., "30 minutes"
	StorePlaintext      bool   `json:"storePlaintext"`
}

// PSO represents a Fine-Grained Password Policy (Password Settings Object)
type PSO struct {
	Name                string   `json:"name"`
	DN                  string   `json:"dn"`
	Precedence          int      `json:"precedence"`
	MinLength           int      `json:"minLength"`
	MaxAge              string   `json:"maxAge"`
	MinAge              string   `json:"minAge"`
	HistoryLength       int      `json:"historyLength"`
	Complexity          bool     `json:"complexity"`
	ReversibleEncryption bool   `json:"reversibleEncryption"`
	LockoutThreshold    int      `json:"lockoutThreshold"`
	LockoutDuration     string   `json:"lockoutDuration"`
	LockoutWindow       string   `json:"lockoutWindow"`
	AppliesTo           []string `json:"appliesTo"`
}

// SearchFilter represents a single filter condition for the visual query builder
type SearchFilter struct {
	Attribute string `json:"attribute"`
	Operator  string `json:"operator"` // equals, contains, startsWith, endsWith, present, notPresent, greaterThan, lessThan, bitwiseAnd, bitwiseOr
	Value     string `json:"value"`
}

// SearchRequest represents an advanced LDAP search request
type SearchRequest struct {
	BaseDN     string         `json:"baseDn"`
	Scope      string         `json:"scope"`      // base, one, sub
	ObjectType string         `json:"objectType"`  // user, group, computer, contact, all
	Filters    []SearchFilter `json:"filters"`
	RawFilter  string         `json:"rawFilter"`   // Raw LDAP filter string (overrides visual filters)
	Attributes []string       `json:"attributes"`  // Attributes to return
}

// SearchResult represents an LDAP search result entry
type SearchResult struct {
	DN         string            `json:"dn"`
	ObjectType string           `json:"objectType"`
	Attributes map[string]string `json:"attributes"`
}

// SavedQuery represents a saved LDAP search query
type SavedQuery struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Request     SearchRequest  `json:"request"`
	CreatedBy   string         `json:"createdBy"`
	CreatedAt   time.Time      `json:"createdAt"`
}

// PasswordTestRequest represents a password test against policy
type PasswordTestRequest struct {
	Password string `json:"password"`
	Username string `json:"username"` // Optional: test against user's effective policy
}

// PasswordTestResult represents the result of testing a password against policy
type PasswordTestResult struct {
	Valid    bool     `json:"valid"`
	Errors  []string `json:"errors"`
	Policy  string   `json:"policy"` // Which policy was tested against (default or PSO name)
}

// GPO represents a Group Policy Object
type GPO struct {
	ID          string   `json:"id"`          // {GUID}
	Name        string   `json:"name"`
	DN          string   `json:"dn"`
	Path        string   `json:"path"`        // sysvol path
	LinksTo     []string `json:"linksTo"`     // OU DNs this GPO is linked to
	Version     int      `json:"version"`
	Flags       int      `json:"flags"`       // 0=enabled, 1=user disabled, 2=computer disabled, 3=all disabled
}

// GPOLink represents a GPO-to-OU link
type GPOLink struct {
	GPOID   string `json:"gpoId"`
	OUDN    string `json:"ouDn"`
	Enabled bool   `json:"enabled"`
}

// SPN represents a Service Principal Name entry
type SPN struct {
	Value   string `json:"value"`   // e.g., "HTTP/myhost.example.com"
	Account string `json:"account"` // sAMAccountName
}

// DelegationInfo represents Kerberos delegation config for an account
type DelegationInfo struct {
	Account         string   `json:"account"`
	Unconstrained   bool     `json:"unconstrained"`
	Constrained     bool     `json:"constrained"`
	AllowedServices []string `json:"allowedServices"`
}

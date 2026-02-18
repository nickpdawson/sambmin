package ldap

// AD attribute name constants used across directory queries.
const (
	// Identity
	AttrDN                = "distinguishedName"
	AttrCN                = "cn"
	AttrSAM               = "sAMAccountName"
	AttrUPN               = "userPrincipalName"
	AttrObjectClass       = "objectClass"
	AttrObjectGUID        = "objectGUID"
	AttrObjectSID         = "objectSid"

	// Display
	AttrDisplayName       = "displayName"
	AttrGivenName         = "givenName"
	AttrSurname           = "sn"
	AttrEmail             = "mail"
	AttrDescription       = "description"

	// Organization
	AttrDepartment        = "department"
	AttrTitle             = "title"
	AttrManager           = "manager"
	AttrCompany           = "company"

	// Account control
	AttrUAC               = "userAccountControl"
	AttrUACComputed       = "msDS-User-Account-Control-Computed"
	AttrAccountExpires    = "accountExpires"
	AttrPwdLastSet        = "pwdLastSet"
	AttrBadPwdCount       = "badPwdCount"
	AttrLockoutTime       = "lockoutTime"

	// Timestamps
	AttrWhenCreated       = "whenCreated"
	AttrWhenChanged       = "whenChanged"
	AttrLastLogon         = "lastLogonTimestamp"

	// Membership
	AttrMemberOf          = "memberOf"
	AttrMember            = "member"

	// Group
	AttrGroupType         = "groupType"

	// Computer
	AttrDNSHostName       = "dnsHostName"
	AttrOS                = "operatingSystem"
	AttrOSVersion         = "operatingSystemVersion"

	// OU
	AttrOU                = "ou"
)

// userAccountControl flag bits
const (
	UACAccountDisable     = 0x0002
	UACLockout            = 0x0010
	UACPasswordNotReq     = 0x0020
	UACNormalAccount      = 0x0200
	UACDontExpirePassword = 0x10000
	UACPasswordExpired    = 0x800000
)

// groupType flag bits
const (
	GroupTypeGlobal       = 0x00000002
	GroupTypeDomainLocal  = 0x00000004
	GroupTypeUniversal    = 0x00000008
	GroupTypeSecurity     = -2147483648 // 0x80000000 (sign bit)
)

// Default attributes to request for each object type (keeps queries efficient).
var UserAttrs = []string{
	AttrDN, AttrSAM, AttrUPN, AttrDisplayName, AttrGivenName, AttrSurname,
	AttrEmail, AttrDepartment, AttrTitle, AttrManager,
	AttrUAC, AttrUACComputed, AttrLockoutTime, AttrPwdLastSet,
	AttrWhenCreated, AttrWhenChanged, AttrLastLogon,
	AttrMemberOf,
}

var GroupAttrs = []string{
	AttrDN, AttrCN, AttrSAM, AttrDescription,
	AttrGroupType, AttrMember, AttrMemberOf,
}

var ComputerAttrs = []string{
	AttrDN, AttrCN, AttrSAM, AttrDNSHostName,
	AttrOS, AttrOSVersion, AttrUAC,
	AttrWhenCreated, AttrLastLogon,
}

var OUAttrs = []string{
	AttrDN, AttrOU, AttrDescription,
}

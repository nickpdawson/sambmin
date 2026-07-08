package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/nickdawson/sambmin/internal/directory"
)

// Delegation of control over AD objects, driven by `samba-tool dsacl`.
//
// Two mechanisms exist and the templates below use whichever fits:
//   - SDDL ACEs (dsacl set --sddl) for object delegations. The trustee is
//     embedded as a SID, so we resolve trustee DN -> objectSid first.
//   - Control Access Rights (dsacl set --car) for the directory-replication
//     rights, which samba only exposes by name. The trustee is a DN.
//
// samba's dsacl only accepts the replication family via --car; everything a
// help-desk or OU admin needs (reset password, create users, manage
// membership, …) has to be expressed as raw SDDL.

// Well-known AD schema GUIDs (stable across AD and Samba).
const (
	guidResetPassword   = "00299570-246d-11d0-a768-00aa006e0529"
	guidChangePassword  = "ab721a53-1e2f-11d0-9819-00aa0040529b"
	guidReplChanges     = "1131f6aa-9c07-11d1-f79f-00c04fc2dcd2"
	guidReplChangesAll  = "1131f6ad-9c07-11d1-f79f-00c04fc2dcd2"
	guidReplChangesFilt = "89e95b76-444d-4c62-991a-0facbeda640c"

	guidClassUser     = "bf967aba-0de6-11d0-a285-00aa003049e2"
	guidClassGroup    = "bf967a9c-0de6-11d0-a285-00aa003049e2"
	guidClassComputer = "bf967a86-0de6-11d0-a285-00aa003049e2"
	guidClassContact  = "5cb41ed0-0e4c-11d0-a286-00aa003049e2"
	guidClassOU       = "bf967aa5-0de6-11d0-a285-00aa003049e2"

	guidAttrMember     = "bf9679c0-0de6-11d0-a285-00aa003049e2"
	guidAttrPwdLastSet = "bf967a0a-0de6-11d0-a285-00aa003049e2"
)

// guidNames maps the schema GUIDs we recognise to display names, for rendering
// existing ACEs.
var guidNames = map[string]string{
	guidResetPassword:   "Reset Password",
	guidChangePassword:  "Change Password",
	guidReplChanges:     "Replicating Directory Changes",
	guidReplChangesAll:  "Replicating Directory Changes (All)",
	guidReplChangesFilt: "Replicating Directory Changes (Filtered Set)",
	guidClassUser:       "user",
	guidClassGroup:      "group",
	guidClassComputer:   "computer",
	guidClassContact:    "contact",
	guidClassOU:         "organizational unit",
	guidAttrMember:      "member",
	guidAttrPwdLastSet:  "pwdLastSet",
}

// rightsNames maps two-letter SDDL access-right codes to short descriptions.
var rightsNames = map[string]string{
	"CC": "create child",
	"DC": "delete child",
	"LC": "list children",
	"SW": "self-write",
	"RP": "read property",
	"WP": "write property",
	"DT": "delete subtree",
	"LO": "list object",
	"CR": "control access",
	"SD": "delete",
	"RC": "read permissions",
	"WD": "modify permissions",
	"WO": "modify owner",
	"GA": "full control",
	"GR": "generic read",
	"GW": "generic write",
	"GX": "generic execute",
}

// delegationTemplate is a named bundle of ACEs (or a --car right) that grants a
// common administrative capability over an OU/subtree.
type delegationTemplate struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Risk        string `json:"risk"`      // low | medium | high
	AppliesTo   string `json:"appliesTo"` // "OU" or "Domain root"

	// car, when set, applies via `dsacl set --car`. Otherwise aces (SDDL with a
	// %SID% placeholder for the trustee) apply via `dsacl set --sddl`.
	car  string
	aces []string
}

// delegationTemplates is the catalogue offered in the UI.
var delegationTemplates = []delegationTemplate{
	{
		Key: "reset_passwords", Label: "Reset user passwords",
		Description: "Reset the password of user accounts in this OU and force a change at next logon. Common for help-desk delegates.",
		Category:    "User accounts", Risk: "medium", AppliesTo: "OU",
		aces: []string{
			"(OA;CI;CR;" + guidResetPassword + ";" + guidClassUser + ";%SID%)",
			"(OA;CI;WP;" + guidAttrPwdLastSet + ";" + guidClassUser + ";%SID%)",
		},
	},
	{
		Key: "manage_user_accounts", Label: "Create, delete, and manage user accounts",
		Description: "Create and delete user objects in this OU and read/write all their properties. Full user lifecycle management.",
		Category:    "User accounts", Risk: "high", AppliesTo: "OU",
		aces: []string{
			"(OA;CI;CCDC;" + guidClassUser + ";;%SID%)",
			"(OA;CI;RPWP;;" + guidClassUser + ";%SID%)",
		},
	},
	{
		Key: "manage_group_membership", Label: "Manage group membership",
		Description: "Add and remove members of groups in this OU (read/write the member attribute).",
		Category:    "Groups", Risk: "medium", AppliesTo: "OU",
		aces: []string{
			"(OA;CI;RPWP;" + guidAttrMember + ";" + guidClassGroup + ";%SID%)",
		},
	},
	{
		Key: "manage_groups", Label: "Create, delete, and manage groups",
		Description: "Create and delete group objects in this OU and read/write all their properties.",
		Category:    "Groups", Risk: "high", AppliesTo: "OU",
		aces: []string{
			"(OA;CI;CCDC;" + guidClassGroup + ";;%SID%)",
			"(OA;CI;RPWP;;" + guidClassGroup + ";%SID%)",
		},
	},
	{
		Key: "join_computers", Label: "Create and delete computer accounts (domain join)",
		Description: "Create and delete computer objects in this OU and manage their properties. Common for a domain-join service account.",
		Category:    "Computers", Risk: "medium", AppliesTo: "OU",
		aces: []string{
			"(OA;CI;CCDC;" + guidClassComputer + ";;%SID%)",
			"(OA;CI;RPWP;;" + guidClassComputer + ";%SID%)",
		},
	},
	{
		Key: "read_all", Label: "Read all objects and properties",
		Description: "Read every object and property in this OU and its subtree. Typical for a read-only bind/service account.",
		Category:    "Read access", Risk: "low", AppliesTo: "OU",
		aces: []string{
			"(A;CI;LCRPRC;;;%SID%)",
		},
	},
	{
		Key: "full_control", Label: "Full control",
		Description: "Complete control over this OU and everything in it, including permissions. Grant sparingly — this is an OU administrator.",
		Category:    "Full control", Risk: "high", AppliesTo: "OU",
		aces: []string{
			"(A;CI;GA;;;%SID%)",
		},
	},
	{
		Key: "replicate_changes", Label: "Replicate directory changes",
		Description: "Read directory changes across the domain (DirSync). Needed by some directory-sync/bind accounts. Apply on the domain root.",
		Category:    "Directory replication", Risk: "high", AppliesTo: "Domain root",
		car: "get-changes",
	},
	{
		Key: "replicate_changes_all", Label: "Replicate directory changes: All (includes secrets)",
		Description: "Read all directory changes including password hashes and secrets (e.g. Azure AD Connect / password-hash sync). Very high privilege — apply on the domain root.",
		Category:    "Directory replication", Risk: "high", AppliesTo: "Domain root",
		car: "get-changes-all",
	},
}

// templateByKey indexes the catalogue.
var templateByKey = func() map[string]delegationTemplate {
	m := make(map[string]delegationTemplate, len(delegationTemplates))
	for _, t := range delegationTemplates {
		m[t.Key] = t
	}
	return m
}()

// aceBodyToTemplate maps an ACE body (everything except the trailing trustee
// field) to the template key that produces it, for attributing existing ACEs.
var aceBodyToTemplate = func() map[string]string {
	m := map[string]string{}
	for _, t := range delegationTemplates {
		for _, ace := range t.aces {
			m[aceBodyKey(ace)] = t.Key
		}
		if t.car != "" {
			// The ACE samba writes for a replication CAR: (OA;;CR;<guid>;;<sid>).
			// Build the body key the same way parseDACL will, so they match.
			guid := map[string]string{"get-changes": guidReplChanges, "get-changes-all": guidReplChangesAll}[t.car]
			if guid != "" {
				m[aceBodyKey("(OA;;CR;"+guid+";;%SID%)")] = t.Key
			}
		}
	}
	// A container-inheritable Generic All (full_control) is canonicalised by the
	// SD layer into two stored ACEs: an inherit-only GA for descendants plus the
	// expanded specific mask on the object itself. Attribute both back to the
	// template so the UI shows one "Full control" grant instead of two rows.
	m[aceBodyKey("(A;CIIO;GA;;;%SID%)")] = "full_control"
	m[aceBodyKey("(A;;"+fullControlMask+";;;%SID%)")] = "full_control"
	return m
}()

// fullControlMask is the specific-rights mask an SD's generic-mapping expands
// Generic All (GA) into when it is stored on a directory object.
const fullControlMask = "CCDCLCSWRPWPDTLOCRSDRCWDWO"

// aceBodyKey returns the ACE with its trailing trustee field replaced by empty,
// so ACEs that differ only by trustee compare equal. Input may be wrapped in
// parens and may use the %SID% placeholder.
func aceBodyKey(ace string) string {
	body := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(ace), "("), ")")
	fields := strings.Split(body, ";")
	if len(fields) < 6 {
		return body
	}
	fields[5] = "" // drop trustee
	return strings.Join(fields[:6], ";")
}

// --- Handlers ---

// handleListDelegationTemplates returns the delegation-template catalogue.
func handleListDelegationTemplates(w http.ResponseWriter, r *http.Request) {
	if requireSession(w, r) == nil {
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"templates": delegationTemplates})
}

// handleGetDSACL reads the ACL on an object and returns the explicit (non-
// inherited) delegations, with trustees resolved to names.
func handleGetDSACL(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}
	objectDN := strings.TrimSpace(r.URL.Query().Get("objectDn"))
	if objectDN == "" {
		respondError(w, http.StatusBadRequest, "objectDn is required")
		return
	}

	out, err := runSambaTool(r.Context(), sess, "dsacl", "get", "--objectdn="+objectDN)
	if err != nil {
		slog.Error("dsacl get failed", "objectDn", objectDN, "actor", sess.Username, "error", err)
		respondSafeError(w, http.StatusInternalServerError, "failed to read access control list", err)
		return
	}

	sddl := extractSDDL(out)
	aces := parseDACL(sddl)

	var sidMap map[string]directory.Trustee
	if dirClient != nil {
		if m, err := dirClient.SIDNameMap(r.Context()); err == nil {
			sidMap = m
		}
	}

	entries := make([]aclEntry, 0, len(aces))
	for _, a := range aces {
		// Show only delegations explicitly set on this object and granted to a
		// real domain principal. That filters out inherited ACEs and the
		// class-default ACEs (SYSTEM, Domain Admins, …) that use well-known
		// aliases or non-domain SIDs — those aren't admin-created delegations.
		if a.inherited || !isDomainPrincipalSID(a.trustee) {
			continue
		}
		key := aceBodyToTemplate[aceBodyKey(a.raw)]
		e := aclEntry{
			TrusteeSID:  a.trustee,
			TrusteeName: directory.TrusteeName(a.trustee, sidMap),
			Rights:      describeACE(a),
			RawAce:      a.raw,
			TemplateKey: key,
		}
		if t, ok := templateByKey[key]; ok {
			e.TemplateLabel = t.Label
		}
		if sidMap != nil {
			if tr, ok := sidMap[a.trustee]; ok {
				e.TrusteeClass = tr.Class
			}
		}
		entries = append(entries, e)
	}

	respondJSON(w, http.StatusOK, map[string]any{"objectDn": objectDN, "entries": entries})
}

type aclEntry struct {
	TrusteeSID    string `json:"trusteeSid"`
	TrusteeName   string `json:"trusteeName"`
	TrusteeClass  string `json:"trusteeClass,omitempty"`
	Rights        string `json:"rights"`
	TemplateKey   string `json:"templateKey,omitempty"`
	TemplateLabel string `json:"templateLabel,omitempty"`
	RawAce        string `json:"rawAce"`
}

type dsaclApplyRequest struct {
	ObjectDN  string   `json:"objectDn"`
	Trustees  []string `json:"trustees"`  // trustee DNs
	Templates []string `json:"templates"` // template keys
}

type dsaclApplyResult struct {
	TrusteeDN     string `json:"trusteeDn"`
	TrusteeName   string `json:"trusteeName"`
	TemplateKey   string `json:"templateKey"`
	TemplateLabel string `json:"templateLabel"`
	OK            bool   `json:"ok"`
	Error         string `json:"error,omitempty"`
}

// handleApplyDSACL applies the selected templates to every selected trustee on
// the target object — the Cartesian product, so a batch of accounts and
// capabilities can be granted in one call.
func handleApplyDSACL(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}
	var req dsaclApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.ObjectDN = strings.TrimSpace(req.ObjectDN)
	if req.ObjectDN == "" || len(req.Trustees) == 0 || len(req.Templates) == 0 {
		respondError(w, http.StatusBadRequest, "objectDn, at least one trustee, and at least one template are required")
		return
	}
	if dirClient == nil {
		respondError(w, http.StatusServiceUnavailable, "directory not available")
		return
	}

	// Validate templates up front.
	for _, k := range req.Templates {
		if _, ok := templateByKey[k]; !ok {
			respondError(w, http.StatusBadRequest, "unknown template: "+k)
			return
		}
	}

	// Resolve each trustee to its SID + name once.
	type trustee struct{ dn, sid, name string }
	resolved := make([]trustee, 0, len(req.Trustees))
	for _, dn := range req.Trustees {
		dn = strings.TrimSpace(dn)
		if dn == "" {
			continue
		}
		sid, err := dirClient.ResolveSID(r.Context(), dn)
		if err != nil {
			respondError(w, http.StatusBadRequest, "cannot resolve trustee "+dn+": "+err.Error())
			return
		}
		name, _ := dirClient.GetSamAccountName(r.Context(), dn)
		resolved = append(resolved, trustee{dn: dn, sid: sid, name: name})
	}

	var results []dsaclApplyResult
	applied, failed := 0, 0
	for _, tr := range resolved {
		for _, key := range req.Templates {
			tmpl := templateByKey[key]
			res := dsaclApplyResult{
				TrusteeDN: tr.dn, TrusteeName: tr.name,
				TemplateKey: key, TemplateLabel: tmpl.Label,
			}
			var err error
			if tmpl.car != "" {
				_, err = runSambaTool(r.Context(), sess, "dsacl", "set",
					"--objectdn="+req.ObjectDN, "--car="+tmpl.car, "--action=allow",
					"--trusteedn="+tr.dn)
			} else {
				sddl := renderACEs(tmpl.aces, tr.sid)
				_, err = runSambaTool(r.Context(), sess, "dsacl", "set",
					"--objectdn="+req.ObjectDN, "--sddl="+sddl)
			}
			if err != nil {
				res.Error = err.Error()
				failed++
				slog.Error("dsacl apply failed", "objectDn", req.ObjectDN, "trustee", tr.dn,
					"template", key, "actor", sess.Username, "error", err)
			} else {
				res.OK = true
				applied++
				slog.Info("delegation applied", "objectDn", req.ObjectDN, "trustee", tr.dn,
					"template", key, "actor", sess.Username)
			}
			results = append(results, res)
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"results": results, "applied": applied, "failed": failed,
	})
}

type dsaclRemoveRequest struct {
	ObjectDN string   `json:"objectDn"`
	SDDLs    []string `json:"sddls"` // exact ACE(s) from a get response
}

// handleRemoveDSACL removes one or more ACEs (each identified by its exact
// SDDL) from an object's ACL. A single delegation can map to several stored
// ACEs (e.g. reset-passwords is two, full-control is two after
// canonicalisation), so the whole group is removed together.
func handleRemoveDSACL(w http.ResponseWriter, r *http.Request) {
	sess := requireSession(w, r)
	if sess == nil {
		return
	}
	var req dsaclRemoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.ObjectDN = strings.TrimSpace(req.ObjectDN)
	if req.ObjectDN == "" || len(req.SDDLs) == 0 {
		respondError(w, http.StatusBadRequest, "objectDn and at least one sddl are required")
		return
	}

	for _, ace := range req.SDDLs {
		ace = strings.TrimSpace(ace)
		if ace == "" {
			continue
		}
		if _, err := runSambaTool(r.Context(), sess, "dsacl", "delete",
			"--objectdn="+req.ObjectDN, "--sddl="+ace); err != nil {
			slog.Error("dsacl delete failed", "objectDn", req.ObjectDN, "ace", ace, "actor", sess.Username, "error", err)
			respondSafeError(w, http.StatusInternalServerError, "failed to remove delegation", err)
			return
		}
	}

	slog.Info("delegation removed", "objectDn", req.ObjectDN, "aces", len(req.SDDLs), "actor", sess.Username)
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

// --- SDDL parsing ---

// isDomainPrincipalSID reports whether an SDDL trustee token is a domain
// principal (user/group/computer), which always has an "S-1-5-21-<domain>-<rid>"
// SID. Well-known aliases (SY, DA, …) and built-in SIDs are not.
func isDomainPrincipalSID(token string) bool {
	return strings.HasPrefix(token, "S-1-5-21-")
}

// extractSDDL pulls the "O:...S:..." descriptor out of dsacl get output, which
// prefixes it with "descriptor for <dn>:".
func extractSDDL(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "O:") {
			return line
		}
	}
	return strings.TrimSpace(out)
}

type parsedACE struct {
	aceType     string
	flags       string
	rights      string
	objectGUID  string
	inheritGUID string
	trustee     string
	inherited   bool
	raw         string // "(...)" as it appeared, for removal
}

// parseDACL extracts the ACEs from the DACL (the "D:" component) of an SDDL
// string. The only colons in an SDDL string are the O:/G:/D:/S: component
// markers, so we can split on them reliably.
func parseDACL(sddl string) []parsedACE {
	dacl := sddlComponent(sddl, 'D')
	if dacl == "" {
		return nil
	}
	var out []parsedACE
	for _, body := range extractParenGroups(dacl) {
		fields := strings.Split(body, ";")
		if len(fields) < 6 {
			continue
		}
		out = append(out, parsedACE{
			aceType:     fields[0],
			flags:       fields[1],
			rights:      fields[2],
			objectGUID:  fields[3],
			inheritGUID: fields[4],
			trustee:     fields[5],
			inherited:   strings.Contains(fields[1], "ID"),
			raw:         "(" + body + ")",
		})
	}
	return out
}

// sddlComponent returns the value of the O/G/D/S component identified by marker.
func sddlComponent(sddl string, marker byte) string {
	// Component starts at "<marker>:" and runs until the next top-level marker
	// (one of O:/G:/D:/S:) or end of string.
	start := -1
	for i := 0; i+1 < len(sddl); i++ {
		if sddl[i] == marker && sddl[i+1] == ':' {
			start = i + 2
			break
		}
	}
	if start < 0 {
		return ""
	}
	end := len(sddl)
	for i := start; i+1 < len(sddl); i++ {
		c := sddl[i]
		if sddl[i+1] == ':' && (c == 'O' || c == 'G' || c == 'D' || c == 'S') {
			// A marker letter is only a real marker when it's not inside an ACE.
			// Component values here are either a SID (no ':') or an ACE list; a
			// bare "<letter>:" can only be the next component, since ACEs contain
			// no colons.
			end = i
			break
		}
	}
	return sddl[start:end]
}

// extractParenGroups returns the contents of each top-level (...) group,
// honouring nesting so conditional ACEs don't split incorrectly.
func extractParenGroups(s string) []string {
	var groups []string
	depth, start := 0, -1
	for i, r := range s {
		switch r {
		case '(':
			if depth == 0 {
				start = i + 1
			}
			depth++
		case ')':
			if depth > 0 {
				depth--
				if depth == 0 && start >= 0 {
					groups = append(groups, s[start:i])
					start = -1
				}
			}
		}
	}
	return groups
}

// renderACEs substitutes the trustee SID into each ACE template and joins them.
func renderACEs(aces []string, sid string) string {
	var sb strings.Builder
	for _, a := range aces {
		sb.WriteString(strings.ReplaceAll(a, "%SID%", sid))
	}
	return sb.String()
}

// describeACE renders a parsed ACE as a short human-readable capability.
func describeACE(a parsedACE) string {
	prefix := ""
	if a.aceType == "D" || a.aceType == "OD" {
		prefix = "DENY: "
	}

	// Extended-right ACE (CR) with a named object GUID.
	if strings.Contains(a.rights, "CR") && a.objectGUID != "" {
		if name, ok := guidNames[a.objectGUID]; ok {
			return prefix + name + scopeSuffix(a)
		}
	}
	if strings.Contains(a.rights, "GA") || a.rights == fullControlMask {
		return prefix + "Full control" + scopeSuffix(a)
	}

	var parts []string
	for _, code := range splitRights(a.rights) {
		if name, ok := rightsNames[code]; ok {
			parts = append(parts, name)
		} else {
			parts = append(parts, code)
		}
	}
	desc := strings.Join(parts, " + ")

	// Attribute / class qualifier.
	if a.objectGUID != "" {
		if name, ok := guidNames[a.objectGUID]; ok {
			desc += " of " + name
		}
	}
	return prefix + desc + scopeSuffix(a)
}

// scopeSuffix describes what descendant class an inherited ACE applies to.
func scopeSuffix(a parsedACE) string {
	if a.inheritGUID == "" {
		return ""
	}
	if name, ok := guidNames[a.inheritGUID]; ok {
		return " (on " + name + " objects)"
	}
	return ""
}

// splitRights breaks an SDDL rights string into its two-letter codes.
func splitRights(rights string) []string {
	var out []string
	for i := 0; i+2 <= len(rights); i += 2 {
		out = append(out, rights[i:i+2])
	}
	return out
}

package handlers

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"

	goldap "github.com/go-ldap/ldap/v3"
)

// handleListSchemaClasses returns AD schema class definitions.
// GET /api/schema/classes
func handleListSchemaClasses(w http.ResponseWriter, r *http.Request) {
	if dirClient == nil {
		respondError(w, http.StatusServiceUnavailable, "directory not available")
		return
	}

	conn, err := dirClient.GetConn(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "ldap connection: "+err.Error())
		return
	}
	defer dirClient.PutConn(conn)

	schemaDN := "CN=Schema,CN=Configuration," + dirClient.BaseDN()

	sr := goldap.NewSearchRequest(
		schemaDN,
		goldap.ScopeSingleLevel,
		goldap.NeverDerefAliases,
		0, 0, false,
		"(objectClass=classSchema)",
		[]string{"cn", "lDAPDisplayName", "adminDescription", "objectClassCategory", "subClassOf", "systemOnly", "defaultObjectCategory"},
		nil,
	)

	result, err := conn.SearchWithPaging(sr, 500)
	if err != nil {
		slog.Error("schema: classes search failed", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list schema classes: "+err.Error())
		return
	}

	type schemaClass struct {
		CN                    string `json:"cn"`
		LDAPDisplayName       string `json:"lDAPDisplayName"`
		Description           string `json:"description"`
		Category              string `json:"category"`
		SubClassOf            string `json:"subClassOf"`
		SystemOnly            bool   `json:"systemOnly"`
		DefaultObjectCategory string `json:"defaultObjectCategory,omitempty"`
	}

	categoryNames := map[string]string{
		"0": "88 Class",
		"1": "Structural",
		"2": "Abstract",
		"3": "Auxiliary",
	}

	var classes []schemaClass
	for _, entry := range result.Entries {
		cat := entry.GetAttributeValue("objectClassCategory")
		catName := categoryNames[cat]
		if catName == "" {
			catName = cat
		}

		classes = append(classes, schemaClass{
			CN:                    entry.GetAttributeValue("cn"),
			LDAPDisplayName:       entry.GetAttributeValue("lDAPDisplayName"),
			Description:           entry.GetAttributeValue("adminDescription"),
			Category:              catName,
			SubClassOf:            entry.GetAttributeValue("subClassOf"),
			SystemOnly:            entry.GetAttributeValue("systemOnly") == "TRUE",
			DefaultObjectCategory: entry.GetAttributeValue("defaultObjectCategory"),
		})
	}

	sort.Slice(classes, func(i, j int) bool {
		return classes[i].LDAPDisplayName < classes[j].LDAPDisplayName
	})

	respondJSON(w, http.StatusOK, map[string]any{
		"classes": classes,
		"total":   len(classes),
	})
}

// handleListSchemaAttributes returns AD schema attribute definitions.
// GET /api/schema/attributes
func handleListSchemaAttributes(w http.ResponseWriter, r *http.Request) {
	if dirClient == nil {
		respondError(w, http.StatusServiceUnavailable, "directory not available")
		return
	}

	conn, err := dirClient.GetConn(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "ldap connection: "+err.Error())
		return
	}
	defer dirClient.PutConn(conn)

	schemaDN := "CN=Schema,CN=Configuration," + dirClient.BaseDN()

	sr := goldap.NewSearchRequest(
		schemaDN,
		goldap.ScopeSingleLevel,
		goldap.NeverDerefAliases,
		0, 0, false,
		"(objectClass=attributeSchema)",
		[]string{"cn", "lDAPDisplayName", "adminDescription", "attributeSyntax", "isSingleValued", "systemOnly", "searchFlags"},
		nil,
	)

	result, err := conn.SearchWithPaging(sr, 500)
	if err != nil {
		slog.Error("schema: attributes search failed", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list schema attributes: "+err.Error())
		return
	}

	// Common AD syntax OIDs to human names
	syntaxNames := map[string]string{
		"2.5.5.1":  "DN",
		"2.5.5.2":  "OID",
		"2.5.5.3":  "Case-Sensitive String",
		"2.5.5.4":  "Case-Insensitive String",
		"2.5.5.5":  "Print Case String",
		"2.5.5.6":  "Numeric String",
		"2.5.5.7":  "DN+Binary",
		"2.5.5.8":  "Boolean",
		"2.5.5.9":  "Integer",
		"2.5.5.10": "Octet String",
		"2.5.5.11": "Generalized Time",
		"2.5.5.12": "Unicode String",
		"2.5.5.13": "Presentation Address",
		"2.5.5.14": "DN+String",
		"2.5.5.15": "NT Security Descriptor",
		"2.5.5.16": "Large Integer",
		"2.5.5.17": "SID",
	}

	type schemaAttr struct {
		CN              string `json:"cn"`
		LDAPDisplayName string `json:"lDAPDisplayName"`
		Description     string `json:"description"`
		Syntax          string `json:"syntax"`
		SyntaxOID       string `json:"syntaxOID"`
		SingleValued    bool   `json:"singleValued"`
		SystemOnly      bool   `json:"systemOnly"`
		Indexed         bool   `json:"indexed"`
	}

	var attrs []schemaAttr
	for _, entry := range result.Entries {
		syntaxOID := entry.GetAttributeValue("attributeSyntax")
		syntaxName := syntaxNames[syntaxOID]
		if syntaxName == "" {
			syntaxName = syntaxOID
		}

		// searchFlags bit 1 = indexed
		searchFlags := entry.GetAttributeValue("searchFlags")
		indexed := false
		if searchFlags != "" && len(searchFlags) > 0 {
			if searchFlags[0] == '1' || searchFlags[0] == '3' || searchFlags[0] == '5' || searchFlags[0] == '7' || searchFlags[0] == '9' {
				indexed = true
			}
		}

		attrs = append(attrs, schemaAttr{
			CN:              entry.GetAttributeValue("cn"),
			LDAPDisplayName: entry.GetAttributeValue("lDAPDisplayName"),
			Description:     entry.GetAttributeValue("adminDescription"),
			Syntax:          syntaxName,
			SyntaxOID:       syntaxOID,
			SingleValued:    entry.GetAttributeValue("isSingleValued") == "TRUE",
			SystemOnly:      entry.GetAttributeValue("systemOnly") == "TRUE",
			Indexed:         indexed,
		})
	}

	sort.Slice(attrs, func(i, j int) bool {
		return attrs[i].LDAPDisplayName < attrs[j].LDAPDisplayName
	})

	// Filter by query param if provided
	q := strings.ToLower(r.URL.Query().Get("q"))
	if q != "" {
		var filtered []schemaAttr
		for _, a := range attrs {
			if strings.Contains(strings.ToLower(a.LDAPDisplayName), q) ||
				strings.Contains(strings.ToLower(a.Description), q) ||
				strings.Contains(strings.ToLower(a.CN), q) {
				filtered = append(filtered, a)
			}
		}
		attrs = filtered
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"attributes": attrs,
		"total":      len(attrs),
	})
}

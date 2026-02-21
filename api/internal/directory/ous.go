package directory

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	goldap "github.com/go-ldap/ldap/v3"

	"github.com/nickdawson/sambmin/internal/ldap"
	"github.com/nickdawson/sambmin/internal/models"
)

// uuidPattern matches GUID-named containers like CN=9738c400-7795-4d6e-b19d-c16cd6486166
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// isSystemContainer returns true for AD internal containers that shouldn't be shown in the OU view.
func isSystemContainer(ou models.OU) bool {
	dn := strings.ToUpper(ou.DN)
	// Skip anything under CN=System
	if strings.Contains(dn, "CN=SYSTEM,") && !strings.HasPrefix(dn, "CN=SYSTEM,") {
		return true
	}
	// Skip the CN=System container itself (but not CN=Users,CN=System)
	if strings.HasPrefix(dn, "CN=SYSTEM,DC=") || strings.HasPrefix(dn, "CN=SYSTEM,DC=") {
		return true
	}
	// Skip GUID-named containers (AD update operations)
	if uuidPattern.MatchString(strings.ToLower(ou.Name)) {
		return true
	}
	// Skip infrastructure containers
	infraContainers := []string{
		"CN=DOMAINUPDATES,", "CN=OPERATIONS,", "CN=WMIPOLICY,",
		"CN=WINSOCKSERVICES,", "CN=IPSEC,", "CN=PROGRAM DATA,",
		"CN=NTDS QUOTAS,", "CN=MANAGED SERVICE ACCOUNTS,",
		"CN=KEYS,", "CN=TPMPOLICY,",
	}
	for _, infra := range infraContainers {
		if strings.Contains(dn, infra) {
			return true
		}
	}
	return false
}

// ListOUs returns all organizational units from the directory.
func (c *Client) ListOUs(ctx context.Context) ([]models.OU, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ous: %w", err)
	}
	defer c.pool.Put(conn)

	sr := goldap.NewSearchRequest(
		c.baseDN,
		goldap.ScopeWholeSubtree,
		goldap.NeverDerefAliases,
		0, 0, false,
		FilterOUs(),
		ldap.OUAttrs,
		nil,
	)

	result, err := conn.SearchWithPaging(sr, 500)
	if err != nil {
		return nil, fmt.Errorf("search ous: %w", err)
	}

	ous := make([]models.OU, 0, len(result.Entries))
	for _, entry := range result.Entries {
		ou := ouFromEntry(entry)
		if isSystemContainer(ou) {
			continue
		}
		// Get child object count via one-level search
		ou.ChildCount = c.countChildren(conn, entry.DN)
		ous = append(ous, ou)
	}

	return ous, nil
}

// GetOU returns a single OU by DN.
func (c *Client) GetOU(ctx context.Context, dn string) (*models.OU, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get ou: %w", err)
	}
	defer c.pool.Put(conn)

	sr := goldap.NewSearchRequest(
		dn,
		goldap.ScopeBaseObject,
		goldap.NeverDerefAliases,
		0, 1, false,
		FilterOUs(),
		ldap.OUAttrs,
		nil,
	)

	result, err := conn.Search(sr)
	if err != nil {
		return nil, fmt.Errorf("search ou %s: %w", dn, err)
	}

	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("ou not found: %s", dn)
	}

	ou := ouFromEntry(result.Entries[0])
	ou.ChildCount = c.countChildren(conn, dn)
	return &ou, nil
}

// GetOUTree returns a hierarchical map of parent DN -> child OUs.
func (c *Client) GetOUTree(ctx context.Context) (map[string][]models.OU, error) {
	ous, err := c.ListOUs(ctx)
	if err != nil {
		return nil, err
	}

	tree := make(map[string][]models.OU)
	for _, ou := range ous {
		if isSystemContainer(ou) {
			continue
		}
		parent := parentDN(ou.DN)
		tree[parent] = append(tree[parent], ou)
	}

	return tree, nil
}

// OUChild represents a single child object in an OU.
type OUChild struct {
	DN          string `json:"dn"`
	Name        string `json:"name"`
	ObjectClass string `json:"objectClass"`
	Description string `json:"description,omitempty"`
}

// ListOUContents returns the direct child objects of an OU.
func (c *Client) ListOUContents(ctx context.Context, dn string) ([]OUChild, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ou contents: %w", err)
	}
	defer c.pool.Put(conn)

	sr := goldap.NewSearchRequest(
		dn,
		goldap.ScopeSingleLevel,
		goldap.NeverDerefAliases,
		0, 0, false,
		"(objectClass=*)",
		[]string{"dn", "name", "objectClass", "description", "sAMAccountName"},
		nil,
	)

	result, err := conn.Search(sr)
	if err != nil {
		return nil, fmt.Errorf("search ou contents: %w", err)
	}

	children := make([]OUChild, 0, len(result.Entries))
	for _, entry := range result.Entries {
		name := entry.GetAttributeValue("name")
		if name == "" {
			name = entry.GetAttributeValue("sAMAccountName")
		}
		if name == "" {
			// Extract CN from DN
			parts := strings.SplitN(entry.DN, ",", 2)
			if len(parts) > 0 {
				name = strings.TrimPrefix(parts[0], "CN=")
				name = strings.TrimPrefix(name, "OU=")
			}
		}

		// Determine primary object class
		classes := entry.GetAttributeValues("objectClass")
		objClass := "object"
		for _, cls := range classes {
			switch cls {
			case "user":
				objClass = "user"
			case "computer":
				if objClass != "user" {
					objClass = "computer"
				}
			case "group":
				if objClass != "user" && objClass != "computer" {
					objClass = "group"
				}
			case "organizationalUnit":
				if objClass == "object" {
					objClass = "ou"
				}
			case "contact":
				if objClass == "object" {
					objClass = "contact"
				}
			case "container":
				if objClass == "object" {
					objClass = "container"
				}
			}
		}

		children = append(children, OUChild{
			DN:          entry.DN,
			Name:        name,
			ObjectClass: objClass,
			Description: entry.GetAttributeValue("description"),
		})
	}

	return children, nil
}

// countChildren returns the number of direct child objects in an OU.
func (c *Client) countChildren(conn *ldap.Conn, dn string) int {
	sr := goldap.NewSearchRequest(
		dn,
		goldap.ScopeSingleLevel,
		goldap.NeverDerefAliases,
		0, 0, false,
		"(objectClass=*)",
		[]string{"dn"},
		nil,
	)
	result, err := conn.Search(sr)
	if err != nil {
		return 0
	}
	return len(result.Entries)
}

// TreeNode represents any object in the directory tree (OU, container, user, group, etc.)
type TreeNode struct {
	DN          string `json:"dn"`
	Name        string `json:"name"`
	ObjectClass string `json:"objectClass"`
	Description string `json:"description,omitempty"`
	ChildCount  int    `json:"childCount,omitempty"`
}

// GetFullTree returns the OU/container tree plus all child objects in each container.
// Returns two maps: the OU tree (parentDN -> []OU) and contents (containerDN -> []TreeNode).
func (c *Client) GetFullTree(ctx context.Context) (map[string][]models.OU, map[string][]TreeNode, error) {
	tree, err := c.GetOUTree(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Collect all container DNs from the tree
	allDNs := make(map[string]bool)
	for _, children := range tree {
		for _, ou := range children {
			allDNs[ou.DN] = true
		}
	}

	// Also include the base DN (domain root) since it contains CN=Users, CN=Computers, etc.
	for parentDN := range tree {
		allDNs[parentDN] = true
	}

	// Fetch contents for each container
	contents := make(map[string][]TreeNode)
	for dn := range allDNs {
		children, err := c.ListOUContents(ctx, dn)
		if err != nil {
			continue // skip containers we can't read
		}
		// Convert OUChild to TreeNode, excluding sub-OUs/containers already in the tree
		subOUDNs := make(map[string]bool)
		for _, sub := range tree[dn] {
			subOUDNs[sub.DN] = true
		}
		var nodes []TreeNode
		for _, child := range children {
			if subOUDNs[child.DN] {
				continue // already shown as a tree branch
			}
			// Skip sub-containers that are already in the OU tree
			if child.ObjectClass == "ou" || child.ObjectClass == "container" {
				if allDNs[child.DN] {
					continue
				}
			}
			nodes = append(nodes, TreeNode{
				DN:          child.DN,
				Name:        child.Name,
				ObjectClass: child.ObjectClass,
				Description: child.Description,
			})
		}
		if len(nodes) > 0 {
			contents[dn] = nodes
		}
	}

	return tree, contents, nil
}

// parentDN returns the parent DN by stripping the first component.
func parentDN(dn string) string {
	// Find first unescaped comma
	for i := 0; i < len(dn); i++ {
		if dn[i] == '\\' {
			i++ // skip escaped char
			continue
		}
		if dn[i] == ',' {
			return dn[i+1:]
		}
	}
	return ""
}

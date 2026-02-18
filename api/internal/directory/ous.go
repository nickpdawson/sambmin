package directory

import (
	"context"
	"fmt"

	goldap "github.com/go-ldap/ldap/v3"

	"github.com/nickdawson/sambmin/internal/ldap"
	"github.com/nickdawson/sambmin/internal/models"
)

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

// GetOUTree returns a hierarchical map of parent DN -> child DNs.
func (c *Client) GetOUTree(ctx context.Context) (map[string][]string, error) {
	ous, err := c.ListOUs(ctx)
	if err != nil {
		return nil, err
	}

	tree := make(map[string][]string)
	for _, ou := range ous {
		parent := parentDN(ou.DN)
		tree[parent] = append(tree[parent], ou.DN)
	}

	return tree, nil
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

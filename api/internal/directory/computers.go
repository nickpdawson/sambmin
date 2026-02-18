package directory

import (
	"context"
	"fmt"

	goldap "github.com/go-ldap/ldap/v3"

	"github.com/nickdawson/sambmin/internal/ldap"
	"github.com/nickdawson/sambmin/internal/models"
)

// ListComputers returns all computer accounts from the directory.
func (c *Client) ListComputers(ctx context.Context, search string) ([]models.Computer, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("list computers: %w", err)
	}
	defer c.pool.Put(conn)

	filter := FilterComputers()
	if search != "" {
		filter = FilterTextSearch(search, filter)
	}

	sr := goldap.NewSearchRequest(
		c.baseDN,
		goldap.ScopeWholeSubtree,
		goldap.NeverDerefAliases,
		0, 0, false,
		filter,
		ldap.ComputerAttrs,
		nil,
	)

	result, err := conn.SearchWithPaging(sr, 500)
	if err != nil {
		return nil, fmt.Errorf("search computers: %w", err)
	}

	computers := make([]models.Computer, 0, len(result.Entries))
	for _, entry := range result.Entries {
		computers = append(computers, computerFromEntry(entry))
	}

	return computers, nil
}

// GetComputer returns a single computer by DN.
func (c *Client) GetComputer(ctx context.Context, dn string) (*models.Computer, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get computer: %w", err)
	}
	defer c.pool.Put(conn)

	sr := goldap.NewSearchRequest(
		dn,
		goldap.ScopeBaseObject,
		goldap.NeverDerefAliases,
		0, 1, false,
		"(objectClass=computer)",
		ldap.ComputerAttrs,
		nil,
	)

	result, err := conn.Search(sr)
	if err != nil {
		return nil, fmt.Errorf("search computer %s: %w", dn, err)
	}

	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("computer not found: %s", dn)
	}

	comp := computerFromEntry(result.Entries[0])
	return &comp, nil
}

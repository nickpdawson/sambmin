package directory

import (
	"context"
	"fmt"

	goldap "github.com/go-ldap/ldap/v3"

	"github.com/nickdawson/sambmin/internal/ldap"
	"github.com/nickdawson/sambmin/internal/models"
)

// ListGroupsOptions configures group listing behavior.
type ListGroupsOptions struct {
	Search       string // text search
	SecurityOnly bool   // only security groups
}

// ListGroups returns groups from the directory.
func (c *Client) ListGroups(ctx context.Context, opts ListGroupsOptions) ([]models.Group, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer c.pool.Put(conn)

	filter := FilterGroups()
	if opts.SecurityOnly {
		filter = FilterSecurityGroups()
	}
	if opts.Search != "" {
		filter = FilterTextSearch(opts.Search, filter)
	}

	sr := goldap.NewSearchRequest(
		c.baseDN,
		goldap.ScopeWholeSubtree,
		goldap.NeverDerefAliases,
		0, 0, false,
		filter,
		ldap.GroupAttrs,
		nil,
	)

	result, err := conn.SearchWithPaging(sr, 500)
	if err != nil {
		return nil, fmt.Errorf("search groups: %w", err)
	}

	groups := make([]models.Group, 0, len(result.Entries))
	for _, entry := range result.Entries {
		groups = append(groups, groupFromEntry(entry))
	}

	return groups, nil
}

// GetGroup returns a single group by DN.
func (c *Client) GetGroup(ctx context.Context, dn string) (*models.Group, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}
	defer c.pool.Put(conn)

	sr := goldap.NewSearchRequest(
		dn,
		goldap.ScopeBaseObject,
		goldap.NeverDerefAliases,
		0, 1, false,
		"(objectClass=group)",
		ldap.GroupAttrs,
		nil,
	)

	result, err := conn.Search(sr)
	if err != nil {
		return nil, fmt.Errorf("search group %s: %w", dn, err)
	}

	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("group not found: %s", dn)
	}

	g := groupFromEntry(result.Entries[0])
	return &g, nil
}

// GetGroupMembers returns the resolved member names for a group.
func (c *Client) GetGroupMembers(ctx context.Context, dn string) ([]models.User, error) {
	group, err := c.GetGroup(ctx, dn)
	if err != nil {
		return nil, err
	}

	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get group members: %w", err)
	}
	defer c.pool.Put(conn)

	var members []models.User
	for _, memberDN := range group.Members {
		sr := goldap.NewSearchRequest(
			memberDN,
			goldap.ScopeBaseObject,
			goldap.NeverDerefAliases,
			0, 1, false,
			"(objectClass=user)",
			ldap.UserAttrs,
			nil,
		)
		result, err := conn.Search(sr)
		if err != nil || len(result.Entries) == 0 {
			continue // Skip non-user members (nested groups, etc.)
		}
		members = append(members, userFromEntry(result.Entries[0]))
	}

	return members, nil
}

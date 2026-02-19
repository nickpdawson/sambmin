package directory

import (
	"context"
	"fmt"

	goldap "github.com/go-ldap/ldap/v3"

	"github.com/nickdawson/sambmin/internal/ldap"
	"github.com/nickdawson/sambmin/internal/models"
)

// ListContacts returns all contacts from the directory, with optional search.
func (c *Client) ListContacts(ctx context.Context, search string) ([]models.Contact, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("list contacts: %w", err)
	}
	defer c.pool.Put(conn)

	filter := FilterContacts()
	if search != "" {
		filter = FilterTextSearch(search, filter)
	}

	sr := goldap.NewSearchRequest(
		c.baseDN,
		goldap.ScopeWholeSubtree,
		goldap.NeverDerefAliases,
		0, 0, false,
		filter,
		ldap.ContactAttrs,
		nil,
	)

	result, err := conn.SearchWithPaging(sr, 500)
	if err != nil {
		return nil, fmt.Errorf("search contacts: %w", err)
	}

	contacts := make([]models.Contact, 0, len(result.Entries))
	for _, entry := range result.Entries {
		contacts = append(contacts, contactFromEntry(entry))
	}

	return contacts, nil
}

// GetContact returns a single contact by DN.
func (c *Client) GetContact(ctx context.Context, dn string) (*models.Contact, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get contact: %w", err)
	}
	defer c.pool.Put(conn)

	sr := goldap.NewSearchRequest(
		dn,
		goldap.ScopeBaseObject,
		goldap.NeverDerefAliases,
		0, 1, false,
		"(objectClass=contact)",
		ldap.ContactAttrs,
		nil,
	)

	result, err := conn.Search(sr)
	if err != nil {
		return nil, fmt.Errorf("search contact %s: %w", dn, err)
	}

	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("contact not found: %s", dn)
	}

	contact := contactFromEntry(result.Entries[0])
	return &contact, nil
}

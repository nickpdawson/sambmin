package dns

import (
	"context"

	"github.com/nickdawson/sambmin/internal/models"
)

// Backend defines the interface for DNS management operations.
// Implementations exist for Samba internal DNS and BIND9 DLZ.
type Backend interface {
	// ListZones returns all DNS zones managed by this backend
	ListZones(ctx context.Context) ([]models.DNSZone, error)

	// ListRecords returns all records in a zone, optionally filtered by type
	ListRecords(ctx context.Context, zone string, recordType string) ([]models.DNSRecord, error)

	// CreateRecord adds a new DNS record to a zone
	CreateRecord(ctx context.Context, zone string, record models.DNSRecord) error

	// UpdateRecord modifies an existing DNS record
	UpdateRecord(ctx context.Context, zone string, name string, record models.DNSRecord) error

	// DeleteRecord removes a DNS record from a zone
	DeleteRecord(ctx context.Context, zone string, name string, recordType string) error

	// CreateZone creates a new DNS zone
	CreateZone(ctx context.Context, zone models.DNSZone) error

	// DeleteZone removes a DNS zone
	DeleteZone(ctx context.Context, zoneName string) error

	// Type returns "samba" or "bind9"
	Type() string
}

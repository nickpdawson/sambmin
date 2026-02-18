package ldap

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	goldap "github.com/go-ldap/ldap/v3"
)

// Pool manages a pool of LDAP connections with multi-DC failover.
type Pool struct {
	mu      sync.Mutex
	dcs     []DCInfo
	baseDN  string
	bindDN  string
	bindPw  string
	useTLS  bool
	conns   chan *Conn
	maxIdle int
	closed  bool
}

// DCInfo describes a domain controller to connect to.
type DCInfo struct {
	Hostname string
	Address  string
	Port     int
	Site     string
	Primary  bool
}

// PoolConfig configures the connection pool.
type PoolConfig struct {
	DCs        []DCInfo
	BaseDN     string
	BindDN     string
	BindPW     string
	UseTLS     bool
	MaxIdle    int
	TLSConfig  *tls.Config
}

// Conn wraps an LDAP connection with metadata.
type Conn struct {
	raw       *goldap.Conn
	dc        string
	createdAt time.Time
	pool      *Pool
	tlsCfg    *tls.Config
}

var poolTLSConfig *tls.Config

// NewPool creates a connection pool for the given DCs.
func NewPool(cfg PoolConfig) (*Pool, error) {
	if len(cfg.DCs) == 0 {
		return nil, fmt.Errorf("ldap pool: no domain controllers configured")
	}
	maxIdle := cfg.MaxIdle
	if maxIdle <= 0 {
		maxIdle = 5
	}

	poolTLSConfig = cfg.TLSConfig
	if poolTLSConfig == nil {
		poolTLSConfig = &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		}
	}

	p := &Pool{
		dcs:     cfg.DCs,
		baseDN:  cfg.BaseDN,
		bindDN:  cfg.BindDN,
		bindPw:  cfg.BindPW,
		useTLS:  cfg.UseTLS,
		conns:   make(chan *Conn, maxIdle),
		maxIdle: maxIdle,
	}

	return p, nil
}

// Get retrieves a connection from the pool or creates a new one.
// Tries the primary DC first, then fails over to secondaries.
func (p *Pool) Get(ctx context.Context) (*Conn, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("ldap pool: closed")
	}
	p.mu.Unlock()

	// Try to reuse an idle connection
	select {
	case c := <-p.conns:
		if c.isHealthy() {
			return c, nil
		}
		c.raw.Close()
	default:
	}

	// Connect to DCs in order (primary first)
	var lastErr error
	for _, dc := range p.dcs {
		c, err := p.dial(ctx, dc)
		if err != nil {
			slog.Warn("ldap: DC unreachable", "dc", dc.Hostname, "error", err)
			lastErr = err
			continue
		}
		return c, nil
	}

	return nil, fmt.Errorf("ldap pool: all DCs unreachable: %w", lastErr)
}

// Put returns a connection to the pool. If the pool is full, the connection is closed.
func (p *Pool) Put(c *Conn) {
	if c == nil {
		return
	}

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		c.raw.Close()
		return
	}
	p.mu.Unlock()

	select {
	case p.conns <- c:
	default:
		c.raw.Close()
	}
}

// Close drains and closes all pooled connections.
func (p *Pool) Close() {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()

	close(p.conns)
	for c := range p.conns {
		c.raw.Close()
	}
}

// BaseDN returns the configured base DN.
func (p *Pool) BaseDN() string {
	return p.baseDN
}

// DCs returns the configured domain controllers.
func (p *Pool) DCs() []DCInfo {
	return p.dcs
}

// dial creates a new connection to a specific DC.
func (p *Pool) dial(ctx context.Context, dc DCInfo) (*Conn, error) {
	addr := fmt.Sprintf("%s:%d", dc.Address, dc.Port)
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	var raw *goldap.Conn
	var err error

	if p.useTLS || dc.Port == 636 {
		tlsCfg := poolTLSConfig.Clone()
		tlsCfg.ServerName = dc.Hostname
		raw, err = goldap.DialURL(
			fmt.Sprintf("ldaps://%s", addr),
			goldap.DialWithTLSConfig(tlsCfg),
			goldap.DialWithDialer(dialer),
		)
	} else {
		raw, err = goldap.DialURL(
			fmt.Sprintf("ldap://%s", addr),
			goldap.DialWithDialer(dialer),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", dc.Hostname, err)
	}

	// Bind with service account if configured
	if p.bindDN != "" {
		if err := raw.Bind(p.bindDN, p.bindPw); err != nil {
			raw.Close()
			return nil, fmt.Errorf("bind to %s: %w", dc.Hostname, err)
		}
	}

	slog.Debug("ldap: connected", "dc", dc.Hostname, "addr", addr)

	return &Conn{
		raw:       raw,
		dc:        dc.Hostname,
		createdAt: time.Now(),
		pool:      p,
		tlsCfg:    poolTLSConfig,
	}, nil
}

// DC returns which domain controller this connection is using.
func (c *Conn) DC() string {
	return c.dc
}

// isHealthy checks if the connection is still alive via a RootDSE search.
func (c *Conn) isHealthy() bool {
	if c.raw == nil {
		return false
	}
	// Quick RootDSE search as health check
	sr := goldap.NewSearchRequest(
		"",
		goldap.ScopeBaseObject,
		goldap.NeverDerefAliases,
		0, 2, false,
		"(objectClass=*)",
		[]string{"namingContexts"},
		nil,
	)
	_, err := c.raw.Search(sr)
	return err == nil
}

// Search performs an LDAP search on this connection.
func (c *Conn) Search(sr *goldap.SearchRequest) (*goldap.SearchResult, error) {
	return c.raw.Search(sr)
}

// SearchWithPaging performs a paged LDAP search to handle AD's 1000-entry limit.
func (c *Conn) SearchWithPaging(sr *goldap.SearchRequest, pageSize uint32) (*goldap.SearchResult, error) {
	return c.raw.SearchWithPaging(sr, pageSize)
}

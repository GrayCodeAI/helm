// Package mcppool provides MCP server pooling for agent sessions.
package mcppool

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// Proxy represents an MCP server proxy
type Proxy struct {
	ID         string
	Name       string
	URL        string
	SocketPath string
	Status     string // healthy, unhealthy, restarting
	LastCheck  time.Time
	Restarts   int
}

// Pool manages a pool of MCP server proxies
type Pool struct {
	mu      sync.RWMutex
	proxies map[string]*Proxy
	health  map[string]bool
}

// NewPool creates a new MCP proxy pool
func NewPool() *Pool {
	return &Pool{
		proxies: make(map[string]*Proxy),
		health:  make(map[string]bool),
	}
}

// AddProxy adds a proxy to the pool
func (p *Pool) AddProxy(proxy *Proxy) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.proxies[proxy.ID] = proxy
	p.health[proxy.ID] = true
}

// RemoveProxy removes a proxy from the pool
func (p *Pool) RemoveProxy(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.proxies, id)
	delete(p.health, id)
}

// GetHealthyProxy returns a healthy proxy by name
func (p *Pool) GetHealthyProxy(name string) (*Proxy, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, proxy := range p.proxies {
		if proxy.Name == name && p.health[proxy.ID] {
			return proxy, nil
		}
	}

	return nil, fmt.Errorf("no healthy proxy found for %s", name)
}

// ListProxies lists all proxies
func (p *Pool) ListProxies() []*Proxy {
	p.mu.RLock()
	defer p.mu.RUnlock()

	proxies := make([]*Proxy, 0, len(p.proxies))
	for _, proxy := range p.proxies {
		proxies = append(proxies, proxy)
	}
	return proxies
}

// CheckHealth checks health of all proxies
func (p *Pool) CheckHealth(ctx context.Context) {
	p.mu.Lock()
	proxyIDs := make([]string, 0, len(p.proxies))
	for id := range p.proxies {
		proxyIDs = append(proxyIDs, id)
	}
	p.mu.Unlock()

	for _, id := range proxyIDs {
		p.mu.RLock()
		proxy := p.proxies[id]
		p.mu.RUnlock()

		healthy := p.checkProxyHealth(ctx, proxy)
		p.mu.Lock()
		p.health[id] = healthy
		proxy.Status = "healthy"
		if !healthy {
			proxy.Status = "unhealthy"
			proxy.Restarts++
		}
		proxy.LastCheck = time.Now()
		p.mu.Unlock()
	}
}

func (p *Pool) checkProxyHealth(ctx context.Context, proxy *Proxy) bool {
	if proxy.URL != "" {
		client := &http.Client{Timeout: 5 * time.Second}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, proxy.URL+"/health", nil)
		resp, err := client.Do(req)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}

	if proxy.SocketPath != "" {
		conn, err := net.DialTimeout("unix", proxy.SocketPath, 5*time.Second)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}

	return false
}

// RestartProxy restarts a failed proxy
func (p *Pool) RestartProxy(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	proxy, ok := p.proxies[id]
	if !ok {
		return fmt.Errorf("proxy not found: %s", id)
	}

	proxy.Status = "restarting"
	proxy.Restarts++

	// In production, would restart the actual proxy process
	proxy.Status = "healthy"
	p.health[id] = true

	return nil
}

// GetStats returns pool statistics
func (p *Pool) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	healthy := 0
	unhealthy := 0
	for _, h := range p.health {
		if h {
			healthy++
		} else {
			unhealthy++
		}
	}

	return map[string]interface{}{
		"total":     len(p.proxies),
		"healthy":   healthy,
		"unhealthy": unhealthy,
	}
}

// Catalog manages MCP tool definitions
type Catalog struct {
	mu    sync.RWMutex
	tools map[string]ToolDef
}

// ToolDef represents an MCP tool definition
type ToolDef struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	Server      string
}

// NewCatalog creates a new tool catalog
func NewCatalog() *Catalog {
	return &Catalog{
		tools: make(map[string]ToolDef),
	}
}

// RegisterTool registers a tool definition
func (c *Catalog) RegisterTool(tool ToolDef) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tools[tool.Name] = tool
}

// GetTool gets a tool by name
func (c *Catalog) GetTool(name string) (ToolDef, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	tool, ok := c.tools[name]
	return tool, ok
}

// ListTools lists all tools
func (c *Catalog) ListTools() []ToolDef {
	c.mu.RLock()
	defer c.mu.RUnlock()
	tools := make([]ToolDef, 0, len(c.tools))
	for _, t := range c.tools {
		tools = append(tools, t)
	}
	return tools
}

// GetToolsByServer gets tools for a specific server
func (c *Catalog) GetToolsByServer(server string) []ToolDef {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var tools []ToolDef
	for _, t := range c.tools {
		if t.Server == server {
			tools = append(tools, t)
		}
	}
	return tools
}

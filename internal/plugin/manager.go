// Package plugin provides plugin system for HELM with dynamic loading.
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"runtime"
	"strings"
	"sync"
)

// Plugin represents a HELM plugin
type Plugin struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Type        string            `json:"type"` // "tool", "skill", "parser", "provider"
	EntryPoint  string            `json:"entry_point"`
	Config      map[string]string `json:"config,omitempty"`
	Enabled     bool              `json:"enabled"`
}

// Manager manages plugins
type Manager struct {
	pluginDir string
	plugins   map[string]*Plugin
	loaded    map[string]*plugin.Plugin
	mu        sync.RWMutex
}

// NewManager creates a plugin manager
func NewManager(pluginDir string) (*Manager, error) {
	os.MkdirAll(pluginDir, 0755)
	return &Manager{
		pluginDir: pluginDir,
		plugins:   make(map[string]*Plugin),
		loaded:    make(map[string]*plugin.Plugin),
	}, nil
}

// Load loads all plugins from the plugin directory
func (m *Manager) Load(ctx context.Context) error {
	entries, err := os.ReadDir(m.pluginDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			manifestPath := filepath.Join(m.pluginDir, entry.Name(), "plugin.json")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				continue
			}

			var p Plugin
			if err := json.Unmarshal(data, &p); err != nil {
				continue
			}

			m.mu.Lock()
			m.plugins[p.Name] = &p
			m.mu.Unlock()
		}
	}

	return nil
}

// LoadSharedLibrary loads a .so plugin file dynamically
func (m *Manager) LoadSharedLibrary(pluginPath string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("plugin loading not supported on Windows")
	}

	// Open the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("open plugin %s: %w", pluginPath, err)
	}

	// Read manifest from plugin directory
	pluginDir := filepath.Dir(pluginPath)
	manifestPath := filepath.Join(pluginDir, "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	var pluginInfo Plugin
	if err := json.Unmarshal(data, &pluginInfo); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	m.mu.Lock()
	m.plugins[pluginInfo.Name] = &pluginInfo
	m.loaded[pluginInfo.Name] = p
	m.mu.Unlock()

	return nil
}

// Install installs a plugin from a URL or local path
func (m *Manager) Install(source string) error {
	var pluginPath string
	var err error

	// Check if source is a URL
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		pluginPath, err = m.downloadPlugin(source)
		if err != nil {
			return fmt.Errorf("download plugin: %w", err)
		}
	} else {
		// Local path - copy to plugin directory
		pluginPath, err = m.copyPlugin(source)
		if err != nil {
			return fmt.Errorf("copy plugin: %w", err)
		}
	}

	// Load the plugin
	if err := m.LoadSharedLibrary(pluginPath); err != nil {
		return fmt.Errorf("load plugin: %w", err)
	}

	return nil
}

// Uninstall removes a plugin
func (m *Manager) Uninstall(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	pluginDir := filepath.Join(m.pluginDir, p.Name)
	if err := os.RemoveAll(pluginDir); err != nil {
		return err
	}

	delete(m.plugins, name)
	delete(m.loaded, name)
	return nil
}

// Enable enables a plugin
func (m *Manager) Enable(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pluginInfo, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	pluginInfo.Enabled = true
	return m.saveManifest(pluginInfo)
}

// Disable disables a plugin
func (m *Manager) Disable(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pluginInfo, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	pluginInfo.Enabled = false
	return m.saveManifest(pluginInfo)
}

// Get gets a plugin by name
func (m *Manager) Get(name string) (*Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pluginInfo, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}
	return pluginInfo, nil
}

// List lists all plugins
func (m *Manager) List() []*Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var plugins []*Plugin
	for _, p := range m.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// ListByType lists plugins by type
func (m *Manager) ListByType(pluginType string) []*Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var plugins []*Plugin
	for _, p := range m.plugins {
		if p.Type == pluginType && p.Enabled {
			plugins = append(plugins, p)
		}
	}
	return plugins
}

// Lookup looks up a symbol in a loaded plugin
func (m *Manager) Lookup(name, symbol string) (plugin.Symbol, error) {
	m.mu.RLock()
	p, exists := m.loaded[name]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("plugin %s not loaded", name)
	}

	sym, err := p.Lookup(symbol)
	if err != nil {
		return nil, fmt.Errorf("lookup symbol %s in %s: %w", symbol, name, err)
	}

	return sym, nil
}

// ExecuteTool executes a tool plugin
func (m *Manager) ExecuteTool(name string, args map[string]interface{}) (interface{}, error) {
	sym, err := m.Lookup(name, "Execute")
	if err != nil {
		return nil, err
	}

	executeFn, ok := sym.(func(map[string]interface{}) (interface{}, error))
	if !ok {
		return nil, fmt.Errorf("invalid Execute function signature in plugin %s", name)
	}

	return executeFn(args)
}

func (m *Manager) saveManifest(p *Plugin) error {
	manifestPath := filepath.Join(m.pluginDir, p.Name, "plugin.json")
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, data, 0644)
}

func (m *Manager) downloadPlugin(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}

	// Determine plugin name from URL
	name := filepath.Base(url)
	name = strings.TrimSuffix(name, ".so")
	name = strings.TrimSuffix(name, ".tar.gz")

	pluginDir := filepath.Join(m.pluginDir, name)
	os.MkdirAll(pluginDir, 0755)

	// For simplicity, assume .so file directly
	// In production, would handle tar.gz extraction
	pluginPath := filepath.Join(pluginDir, name+".so")

	out, err := os.Create(pluginPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := out.ReadFrom(resp.Body); err != nil {
		return "", err
	}

	return pluginPath, nil
}

func (m *Manager) copyPlugin(source string) (string, error) {
	name := filepath.Base(source)
	name = strings.TrimSuffix(name, ".so")

	pluginDir := filepath.Join(m.pluginDir, name)
	os.MkdirAll(pluginDir, 0755)

	pluginPath := filepath.Join(pluginDir, name+".so")

	// Use cp command for simplicity
	cmd := exec.Command("cp", source, pluginPath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("copy plugin: %w", err)
	}

	return pluginPath, nil
}

// Loader loads plugin code
type Loader struct {
	manager *Manager
}

// NewLoader creates a plugin loader
func NewLoader(manager *Manager) *Loader {
	return &Loader{manager: manager}
}

// LoadTool loads a tool plugin
func (l *Loader) LoadTool(name string) (ToolPlugin, error) {
	_, err := l.manager.Get(name)
	if err != nil {
		return nil, err
	}

	sym, err := l.manager.Lookup(name, "NewTool")
	if err != nil {
		return nil, err
	}

	newToolFn, ok := sym.(func() ToolPlugin)
	if !ok {
		return nil, fmt.Errorf("invalid NewTool function signature in plugin %s", name)
	}

	return newToolFn(), nil
}

// ToolPlugin is the interface for tool plugins
type ToolPlugin interface {
	Name() string
	Execute(args map[string]interface{}) (interface{}, error)
}

// SkillPlugin is the interface for skill plugins
type SkillPlugin interface {
	Name() string
	Prompt() string
	Context() map[string]string
}

// Marketplace represents a plugin marketplace
type Marketplace struct {
	url string
}

// NewMarketplace creates a marketplace client
func NewMarketplace(url string) *Marketplace {
	return &Marketplace{url: url}
}

// Search searches for plugins in the marketplace
func (m *Marketplace) Search(query string) ([]Plugin, error) {
	if m.url == "" {
		return []Plugin{}, nil
	}

	resp, err := http.Get(fmt.Sprintf("%s/search?q=%s", m.url, query))
	if err != nil {
		return nil, fmt.Errorf("search marketplace: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace error: %s", resp.Status)
	}

	var plugins []Plugin
	if err := json.NewDecoder(resp.Body).Decode(&plugins); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return plugins, nil
}

// Download downloads a plugin from the marketplace
func (m *Marketplace) Download(name, version string) (string, error) {
	if m.url == "" {
		return "", fmt.Errorf("no marketplace URL configured")
	}

	url := fmt.Sprintf("%s/plugins/%s/%s/download", m.url, name, version)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("download plugin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}

	// Save to temp file
	tmpDir, err := os.MkdirTemp("", "helm-plugin-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	pluginPath := filepath.Join(tmpDir, name+".so")
	out, err := os.Create(pluginPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := out.ReadFrom(resp.Body); err != nil {
		return "", err
	}

	return pluginPath, nil
}

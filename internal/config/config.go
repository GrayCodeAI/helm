// Package config provides HELM configuration management with validation and hot-reload.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

// Config holds all HELM configuration.
type Config struct {
	mu sync.RWMutex

	// Core settings
	AppName   string `toml:"app_name" env:"HELM_APP_NAME" default:"helm"`
	Version   string `toml:"version" env:"HELM_VERSION" default:"0.1.0"`
	LogLevel  string `toml:"log_level" env:"HELM_LOG_LEVEL" default:"info"`
	LogFormat string `toml:"log_format" env:"HELM_LOG_FORMAT" default:"text"`
	DataDir   string `toml:"data_dir" env:"HELM_DATA_DIR" default:"~/.local/share/helm"`

	// Provider settings
	Providers ProvidersConfig `toml:"providers"`
	Router    RouterConfig    `toml:"router"`

	// Feature settings
	Budget  BudgetConfig  `toml:"budget"`
	UI      UIConfig      `toml:"ui"`
	Session SessionConfig `toml:"session"`
	Memory  MemoryConfig  `toml:"memory"`

	// Server settings
	Server ServerConfig `toml:"server"`

	// Metrics
	Metrics MetricsConfig `toml:"metrics"`

	// Hot reload support
	reloadCallbacks []func()
	watchers        []chan struct{}
}

// ProvidersConfig holds provider-specific settings.
type ProvidersConfig struct {
	Anthropic  ProviderConfig `toml:"anthropic"`
	OpenAI     ProviderConfig `toml:"openai"`
	Google     ProviderConfig `toml:"google"`
	Ollama     OllamaConfig   `toml:"ollama"`
	OpenRouter ProviderConfig `toml:"openrouter"`
}

// ProviderConfig is a generic provider configuration.
type ProviderConfig struct {
	APIKey       string        `toml:"api_key" env:"API_KEY"`
	DefaultModel string        `toml:"default_model"`
	BaseURL      string        `toml:"base_url,omitempty"`
	Timeout      time.Duration `toml:"timeout" default:"60s"`
	MaxRetries   int           `toml:"max_retries" default:"3"`
}

// OllamaConfig holds Ollama-specific settings.
type OllamaConfig struct {
	BaseURL      string `toml:"base_url"`
	DefaultModel string `toml:"default_model"`
}

// RouterConfig holds provider routing settings.
type RouterConfig struct {
	FallbackChain  []string `toml:"fallback_chain"`
	RateLimitRetry bool     `toml:"rate_limit_retry"`
	MaxRetries     int      `toml:"max_retries"`
}

// BudgetConfig holds budget settings.
type BudgetConfig struct {
	DailyLimit    float64 `toml:"daily_limit"`
	WeeklyLimit   float64 `toml:"weekly_limit"`
	MonthlyLimit  float64 `toml:"monthly_limit"`
	WarningPct    float64 `toml:"warning_pct"`
	ActionOnLimit string  `toml:"action_on_limit"`
	Enabled       bool    `toml:"enabled" default:"true"`
}

// UIConfig holds UI settings.
type UIConfig struct {
	Theme            string `toml:"theme"`
	ShowCostInStatus bool   `toml:"show_cost_in_status"`
	CompactMode      bool   `toml:"compact_mode" default:"false"`
}

// SessionConfig holds session settings.
type SessionConfig struct {
	AutoSave         bool          `toml:"auto_save" default:"true"`
	ArchiveAfterDays int           `toml:"archive_after_days" default:"30"`
	MaxConcurrent    int           `toml:"max_concurrent" default:"5"`
	DefaultTimeout   time.Duration `toml:"default_timeout" default:"30m"`
}

// MemoryConfig holds memory settings.
type MemoryConfig struct {
	Enabled         bool    `toml:"enabled" default:"true"`
	AutoLearn       bool    `toml:"auto_learn" default:"true"`
	MaxEntries      int     `toml:"max_entries" default:"1000"`
	MinConfidence   float64 `toml:"min_confidence" default:"0.5"`
	ForgetAfterDays int     `toml:"forget_after_days" default:"90"`
}

// ServerConfig holds server settings.
type ServerConfig struct {
	Enabled      bool          `toml:"enabled" default:"true"`
	Host         string        `toml:"host" default:"127.0.0.1"`
	Port         int           `toml:"port" default:"8080"`
	ReadTimeout  time.Duration `toml:"read_timeout" default:"30s"`
	WriteTimeout time.Duration `toml:"write_timeout" default:"30s"`
	EnableCORS   bool          `toml:"enable_cors" default:"true"`
	EnableAuth   bool          `toml:"enable_auth" default:"false"`
}

// MetricsConfig holds metrics settings.
type MetricsConfig struct {
	Enabled bool   `toml:"enabled" default:"false"`
	Port    int    `toml:"port" default:"9090"`
	Path    string `toml:"path" default:"/metrics"`
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		AppName:   "helm",
		Version:   "0.1.0",
		LogLevel:  "info",
		LogFormat: "text",
		DataDir:   "~/.local/share/helm",
		Providers: ProvidersConfig{
			Anthropic: ProviderConfig{
				DefaultModel: "claude-sonnet-4-20250514",
				Timeout:      60 * time.Second,
				MaxRetries:   3,
			},
			OpenAI: ProviderConfig{
				DefaultModel: "gpt-4o",
				Timeout:      60 * time.Second,
				MaxRetries:   3,
			},
			Google: ProviderConfig{
				DefaultModel: "gemini-2.5-pro",
				Timeout:      60 * time.Second,
				MaxRetries:   3,
			},
			Ollama: OllamaConfig{
				BaseURL:      "http://localhost:11434",
				DefaultModel: "qwen2.5-coder:32b",
			},
			OpenRouter: ProviderConfig{
				DefaultModel: "anthropic/claude-sonnet-4",
				Timeout:      60 * time.Second,
				MaxRetries:   3,
			},
		},
		Router: RouterConfig{
			FallbackChain:  []string{"anthropic", "openai", "openrouter"},
			RateLimitRetry: true,
			MaxRetries:     3,
		},
		Budget: BudgetConfig{
			DailyLimit:    10.0,
			WeeklyLimit:   50.0,
			MonthlyLimit:  150.0,
			WarningPct:    0.80,
			ActionOnLimit: "pause",
			Enabled:       true,
		},
		UI: UIConfig{
			Theme:            "default",
			ShowCostInStatus: true,
			CompactMode:      false,
		},
		Session: SessionConfig{
			AutoSave:         true,
			ArchiveAfterDays: 30,
			MaxConcurrent:    5,
			DefaultTimeout:   30 * time.Minute,
		},
		Memory: MemoryConfig{
			Enabled:         true,
			AutoLearn:       true,
			MaxEntries:      1000,
			MinConfidence:   0.5,
			ForgetAfterDays: 90,
		},
		Server: ServerConfig{
			Enabled:      true,
			Host:         "127.0.0.1",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			EnableCORS:   true,
			EnableAuth:   false,
		},
		Metrics: MetricsConfig{
			Enabled: false,
			Port:    9090,
			Path:    "/metrics",
		},
	}
}

// Load reads configuration from the given path.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	// Load environment variables
	cfg.loadEnvVars()

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadEnvVars loads API keys from environment variables.
func (c *Config) loadEnvVars() {
	// Provider API keys
	if c.Providers.Anthropic.APIKey == "" {
		c.Providers.Anthropic.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if c.Providers.OpenAI.APIKey == "" {
		c.Providers.OpenAI.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	if c.Providers.Google.APIKey == "" {
		c.Providers.Google.APIKey = os.Getenv("GOOGLE_API_KEY")
	}
	if c.Providers.OpenRouter.APIKey == "" {
		c.Providers.OpenRouter.APIKey = os.Getenv("OPENROUTER_API_KEY")
	}

	// Other env vars
	if v := os.Getenv("HELM_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("HELM_DATA_DIR"); v != "" {
		c.DataDir = v
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	// Validate log level
	validLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLevels, c.LogLevel) {
		return fmt.Errorf("invalid log_level: %s", c.LogLevel)
	}

	// Validate server port
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Validate budget
	if c.Budget.DailyLimit < 0 {
		return fmt.Errorf("daily budget must be non-negative")
	}

	return nil
}

// Save writes configuration to the given path.
func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	return enc.Encode(c)
}

// ConfigPath returns the default config path.
func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".helm/helm.toml"
	}
	return filepath.Join(home, ".helm", "helm.toml")
}

// ProjectConfigPath returns the project-local config path.
func ProjectConfigPath() string {
	return ".helm/helm.toml"
}

// Get returns a config value by path (e.g., "server.port")
func (c *Config) Get(path string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	v := reflect.ValueOf(c).Elem()

	for _, part := range parts {
		field := findField(v, part)
		if !field.IsValid() {
			return nil, fmt.Errorf("config key not found: %s", path)
		}
		v = field
	}

	return v.Interface(), nil
}

// Set sets a config value by path
func (c *Config) Set(path string, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return fmt.Errorf("empty path")
	}

	v := reflect.ValueOf(c).Elem()

	for i, part := range parts {
		field := findField(v, part)
		if !field.IsValid() {
			return fmt.Errorf("config key not found: %s", path)
		}

		if i == len(parts)-1 {
			if err := setField(field, value); err != nil {
				return err
			}
		} else {
			v = field
		}
	}

	// Notify watchers
	c.notifyReload()

	return nil
}

// OnReload registers a callback for config reload
func (c *Config) OnReload(fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reloadCallbacks = append(c.reloadCallbacks, fn)
}

// Watch returns a channel that receives updates on reload
func (c *Config) Watch() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := make(chan struct{}, 1)
	c.watchers = append(c.watchers, ch)
	return ch
}

func (c *Config) notifyReload() {
	for _, fn := range c.reloadCallbacks {
		go fn()
	}

	for _, ch := range c.watchers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// Helper functions

func findField(v reflect.Value, name string) reflect.Value {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		if strings.EqualFold(fieldType.Name, name) {
			return field
		}

		if tag := fieldType.Tag.Get("toml"); tag != "" {
			if strings.Split(tag, ",")[0] == name {
				return field
			}
		}
	}

	return reflect.Value{}
}

func setField(field reflect.Value, value interface{}) error {
	if value == nil {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(fmt.Sprint(value))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v, ok := value.(int); ok {
			field.SetInt(int64(v))
		} else if s, ok := value.(string); ok {
			if i, err := strconv.ParseInt(s, 10, 64); err == nil {
				field.SetInt(i)
			}
		}
	case reflect.Bool:
		if v, ok := value.(bool); ok {
			field.SetBool(v)
		} else if s, ok := value.(string); ok {
			if b, err := strconv.ParseBool(s); err == nil {
				field.SetBool(b)
			}
		}
	case reflect.Float64:
		if v, ok := value.(float64); ok {
			field.SetFloat(v)
		} else if s, ok := value.(string); ok {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				field.SetFloat(f)
			}
		}
	case reflect.Struct:
		// Handle time.Duration
		if field.Type().String() == "time.Duration" {
			if s, ok := value.(string); ok {
				if d, err := time.ParseDuration(s); err == nil {
					field.Set(reflect.ValueOf(d))
				}
			}
		}
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Global config instance
var (
	globalConfig *Config
	configMu     sync.RWMutex
)

// SetGlobal sets the global config
func SetGlobal(cfg *Config) {
	configMu.Lock()
	defer configMu.Unlock()
	globalConfig = cfg
}

// GetGlobal returns the global config
func GetGlobal() *Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return globalConfig
}

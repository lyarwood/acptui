package ambient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

type Config struct {
	APIUrl            string `json:"api_url,omitempty"`
	FrontendUrl       string `json:"frontend_url,omitempty"`
	AccessToken       string `json:"access_token,omitempty"`
	Project           string `json:"project,omitempty"`
	User              string `json:"user,omitempty"`
	InsecureTLSVerify bool   `json:"insecure_tls_verify,omitempty"`
}

func ConfigLocation() (string, error) {
	if env := os.Getenv("AMBIENT_CONFIG"); env != "" {
		return env, nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("determine config directory: %w", err)
	}
	return filepath.Join(configDir, "ambient", "config.json"), nil
}

func LoadConfig() (*Config, error) {
	location, err := ConfigLocation()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(location)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read config file %q: %w", location, err)
	}
	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config file %q: %w", location, err)
	}
	return cfg, nil
}

func SaveConfig(cfg *Config) error {
	location, err := ConfigLocation()
	if err != nil {
		return err
	}
	dir := filepath.Dir(location)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(location, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func (c *Config) GetAPIUrl() string {
	if env := os.Getenv("AMBIENT_API_URL"); env != "" {
		return env
	}
	if c.APIUrl != "" {
		return c.APIUrl
	}
	return "http://localhost:8000"
}

func (c *Config) GetProject() string {
	if env := os.Getenv("AMBIENT_PROJECT"); env != "" {
		return env
	}
	return c.Project
}

func (c *Config) GetToken() string {
	if env := os.Getenv("AMBIENT_TOKEN"); env != "" {
		return env
	}
	return c.AccessToken
}

func (c *Config) GetUser() string {
	if env := os.Getenv("AMBIENT_USER"); env != "" {
		return env
	}
	return c.User
}

func (c *Config) GetFrontendUrl() string {
	if env := os.Getenv("AMBIENT_FRONTEND_URL"); env != "" {
		return env
	}
	if c.FrontendUrl != "" {
		return c.FrontendUrl
	}
	if c.APIUrl != "" {
		return c.APIUrl
	}
	return "http://localhost:3000"
}

type ClientConfig struct {
	FrontendURL string
	apiURL      string
	token       string
	user        string
	insecureTLS bool
}

func (cc *ClientConfig) ProviderForProject(project string) (*Provider, error) {
	return NewProvider(cc.apiURL, cc.token, cc.user, project, cc.insecureTLS), nil
}

func NewClientFromConfig(version string, insecureTLS bool) (*ClientConfig, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	token := cfg.GetToken()
	if token == "" {
		return nil, fmt.Errorf("not logged in; run 'acpctl login' first")
	}

	apiURL := cfg.GetAPIUrl()
	parsed, err := url.Parse(apiURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid API URL %q: must include scheme and host (e.g. https://api.example.com)", apiURL)
	}

	return &ClientConfig{
		FrontendURL: cfg.GetFrontendUrl(),
		apiURL:      apiURL,
		token:       token,
		user:        cfg.GetUser(),
		insecureTLS: cfg.InsecureTLSVerify || insecureTLS,
	}, nil
}

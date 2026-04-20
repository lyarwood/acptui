package cmd

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/lyarwood/acptui/internal/ambient"
)

var loginCmd = &cobra.Command{
	Use:   "login [URL]",
	Short: "Log in to an Ambient Code Platform instance via OpenShift OAuth",
	Long: `Authenticate with the Ambient Code Platform by logging in through OpenShift OAuth.

Opens your browser to the OpenShift login page. After authentication, you'll
receive a token to paste back into the terminal.

The URL should be the frontend URL of your Ambient Code Platform instance
(e.g. https://ambient-code.apps.example.com).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := ambient.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		apiURL := ""
		if len(args) > 0 {
			apiURL = strings.TrimRight(args[0], "/")
		} else if cfg.GetAPIUrl() != "http://localhost:8000" {
			apiURL = strings.TrimRight(cfg.GetAPIUrl(), "/")
		}
		if apiURL == "" {
			return fmt.Errorf("URL required: acptui login https://ambient-code.apps.example.com")
		}

		// Discover the OAuth server from the frontend URL's apps domain
		oauthURL, err := discoverOAuthServer(apiURL)
		if err != nil {
			return fmt.Errorf("discovering OAuth server: %w", err)
		}

		tokenRequestURL := oauthURL + "/oauth/token/request"
		fmt.Printf("Opening browser for authentication...\n")
		fmt.Printf("If the browser doesn't open, visit:\n  %s\n\n", tokenRequestURL)

		openBrowserLogin(tokenRequestURL)

		fmt.Print("Paste your token here: ")
		reader := bufio.NewReader(os.Stdin)
		token, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading token: %w", err)
		}
		token = strings.TrimSpace(token)
		if token == "" {
			return fmt.Errorf("no token provided")
		}

		// Validate the token by calling /api/me
		user, err := validateToken(apiURL, token)
		if err != nil {
			return fmt.Errorf("token validation failed: %w", err)
		}

		// Update config
		cfg.APIUrl = apiURL
		cfg.AccessToken = token
		if user != "" {
			cfg.User = user
		}

		if err := ambient.SaveConfig(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("Logged in as %s\n", user)
		fmt.Printf("API URL: %s\n", apiURL)
		if cfg.Project != "" {
			fmt.Printf("Project: %s\n", cfg.Project)
		} else {
			fmt.Println("No project set. Use: acptui login and set project in config, or set AMBIENT_PROJECT")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

// discoverOAuthServer finds the OAuth server URL by following the frontend's
// /oauth/start redirect, which reveals the OAuth server hostname.
func discoverOAuthServer(frontendURL string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}, //nolint:gosec
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(frontendURL + "/oauth/start")
	if err != nil {
		return "", fmt.Errorf("failed to reach %s/oauth/start: %w", frontendURL, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusTemporaryRedirect {
		// No OAuth proxy — try deriving from the apps domain
		return deriveOAuthURL(frontendURL)
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("no redirect from /oauth/start")
	}

	u, err := url.Parse(location)
	if err != nil {
		return "", fmt.Errorf("parse redirect URL: %w", err)
	}

	return fmt.Sprintf("%s://%s", u.Scheme, u.Host), nil
}

func deriveOAuthURL(frontendURL string) (string, error) {
	u, err := url.Parse(frontendURL)
	if err != nil {
		return "", err
	}

	// Extract apps domain: ambient-code.apps.rosa.example.com → apps.rosa.example.com
	host := u.Hostname()
	idx := strings.Index(host, ".apps.")
	if idx < 0 {
		idx = strings.Index(host, ".apps-")
	}
	if idx < 0 {
		return "", fmt.Errorf("cannot derive OAuth URL from %s — not an OpenShift apps domain", host)
	}
	appsDomain := host[idx+1:]

	return fmt.Sprintf("https://oauth-openshift.%s", appsDomain), nil
}

func validateToken(apiURL, token string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}, //nolint:gosec
		},
	}

	req, err := http.NewRequest(http.MethodGet, apiURL+"/api/me", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Forwarded-Access-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var me struct {
		Authenticated bool   `json:"authenticated"`
		UserID        string `json:"userId"`
		Username      string `json:"username"`
		DisplayName   string `json:"displayName"`
	}
	if err := json.Unmarshal(body, &me); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if !me.Authenticated {
		return "", fmt.Errorf("token is not authenticated")
	}

	user := me.Username
	if user == "" {
		user = me.UserID
	}
	return user, nil
}

func openBrowserLogin(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	}
	if err != nil {
		// Browser open is best-effort
		_ = err
	}
}

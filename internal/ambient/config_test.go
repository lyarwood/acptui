package ambient_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/lyarwood/acptui/internal/ambient"
)

var _ = Describe("Config", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "acptui-config-test")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		_ = os.RemoveAll(tmpDir)
		_ = os.Unsetenv("AMBIENT_CONFIG")
		_ = os.Unsetenv("AMBIENT_API_URL")
		_ = os.Unsetenv("AMBIENT_TOKEN")
		_ = os.Unsetenv("AMBIENT_PROJECT")
	})

	Describe("LoadConfig", func() {
		It("returns empty config when file does not exist", func() {
			_ = os.Setenv("AMBIENT_CONFIG", filepath.Join(tmpDir, "nonexistent.json"))
			cfg, err := ambient.LoadConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).NotTo(BeNil())
		})

		It("parses a valid config file", func() {
			configPath := filepath.Join(tmpDir, "config.json")
			err := os.WriteFile(configPath, []byte(`{
				"api_url": "https://api.example.com",
				"access_token": "test-token-that-is-long-enough",
				"project": "my-project"
			}`), 0600)
			Expect(err).NotTo(HaveOccurred())

			_ = os.Setenv("AMBIENT_CONFIG", configPath)
			cfg, err := ambient.LoadConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.GetAPIUrl()).To(Equal("https://api.example.com"))
			Expect(cfg.GetToken()).To(Equal("test-token-that-is-long-enough"))
			Expect(cfg.GetProject()).To(Equal("my-project"))
		})

		It("returns error for invalid JSON", func() {
			configPath := filepath.Join(tmpDir, "config.json")
			err := os.WriteFile(configPath, []byte(`{invalid`), 0600)
			Expect(err).NotTo(HaveOccurred())

			_ = os.Setenv("AMBIENT_CONFIG", configPath)
			_, err = ambient.LoadConfig()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("environment variable overrides", func() {
		It("overrides API URL from env", func() {
			_ = os.Setenv("AMBIENT_CONFIG", filepath.Join(tmpDir, "nonexistent.json"))
			_ = os.Setenv("AMBIENT_API_URL", "https://env-api.example.com")

			cfg, err := ambient.LoadConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.GetAPIUrl()).To(Equal("https://env-api.example.com"))
		})

		It("overrides token from env", func() {
			_ = os.Setenv("AMBIENT_CONFIG", filepath.Join(tmpDir, "nonexistent.json"))
			_ = os.Setenv("AMBIENT_TOKEN", "env-token-value")

			cfg, err := ambient.LoadConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.GetToken()).To(Equal("env-token-value"))
		})

		It("overrides project from env", func() {
			_ = os.Setenv("AMBIENT_CONFIG", filepath.Join(tmpDir, "nonexistent.json"))
			_ = os.Setenv("AMBIENT_PROJECT", "env-project")

			cfg, err := ambient.LoadConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.GetProject()).To(Equal("env-project"))
		})

		It("defaults API URL to localhost", func() {
			_ = os.Setenv("AMBIENT_CONFIG", filepath.Join(tmpDir, "nonexistent.json"))

			cfg, err := ambient.LoadConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.GetAPIUrl()).To(Equal("http://localhost:8000"))
		})
	})
})

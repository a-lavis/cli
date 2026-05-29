package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/buildkite/cli/v3/internal/configfile"
	"github.com/buildkite/cli/v3/pkg/keyring"
	git "github.com/go-git/go-git/v5"
	"github.com/spf13/afero"
)

var (
	legacyTokenWarningOnce sync.Once
	envTokenWarningOnce    sync.Once
)

const (
	DefaultGraphQLEndpoint = configfile.DefaultGraphQLEndpoint
	ExperimentPreflight    = configfile.ExperimentPreflight
	DefaultExperiments     = configfile.DefaultExperiments
)

// Config resolves effective CLI configuration and credentials.
//
// It embeds configfile.Config so all file-backed configuration methods remain
// available while adding token resolution against keyring/environment.
type Config struct {
	*configfile.Config
}

func New(fs afero.Fs, repo *git.Repository) *Config {
	base := configfile.New(fs, repo)
	return &Config{Config: base}
}

// APIToken gets the API token configured for the currently selected organization.
// Precedence: environment variable > keyring > config file (legacy, read-only with warning)
func (conf *Config) APIToken() string {
	return conf.APITokenForOrg(conf.OrganizationSlug())
}

// APITokenForOrg gets the API token for a specific organization.
// Precedence: environment variable > keyring > config file (legacy, read-only with warning)
func (conf *Config) APITokenForOrg(org string) string {
	if token := os.Getenv("BUILDKITE_API_TOKEN"); token != "" {
		envTokenWarningOnce.Do(func() {
			fmt.Fprintln(os.Stderr, "Warning: using BUILDKITE_API_TOKEN environment variable for authentication.")
		})
		return token
	}

	kr := keyring.New()
	if kr.IsAvailable() {
		if token, err := kr.Get(org); err == nil && token != "" {
			return token
		}
	}

	if token := conf.LegacyTokenForOrg(org); token != "" {
		legacyTokenWarningOnce.Do(func() {
			fmt.Fprintln(os.Stderr, "Warning: reading API token from config file is deprecated. Run `bk auth login` to store your token securely in the system keychain.")
		})
		return token
	}

	return ""
}

// RefreshTokenForOrg gets the refresh token for a specific organization from the keyring.
func (conf *Config) RefreshTokenForOrg(org string) string {
	if org == "" {
		return ""
	}

	kr := keyring.New()
	if kr.IsAvailable() {
		if token, err := kr.GetRefreshToken(org); err == nil && token != "" {
			return token
		}
	}
	return ""
}

// RefreshToken gets the refresh token for the currently selected organization.
func (conf *Config) RefreshToken() string {
	return conf.RefreshTokenForOrg(conf.OrganizationSlug())
}

// HasStoredTokenForOrg reports whether a token is stored for org in keyring
// or config files, excluding environment variable overrides.
func (conf *Config) HasStoredTokenForOrg(org string) bool {
	if org == "" {
		return false
	}

	kr := keyring.New()
	if kr.IsAvailable() {
		if token, err := kr.Get(org); err == nil && token != "" {
			return true
		}
	}
	return conf.HasLegacyTokenForOrg(org)
}

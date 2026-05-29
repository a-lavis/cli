package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/buildkite/cli/v3/pkg/keyring"
)

func setEnvCredentialTest(t *testing.T, key, value string) {
	t.Helper()
	original, had := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env %s: %v", key, err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, original)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func TestConfig_APITokenForOrgReadsLegacyWhenKeyringDisabled(t *testing.T) {
	setEnvCredentialTest(t, "BUILDKITE_NO_KEYRING", "1")
	setEnvCredentialTest(t, "CI", "")
	setEnvCredentialTest(t, "BUILDKITE", "")
	setEnvCredentialTest(t, "BUILDKITE_API_TOKEN", "")
	keyring.ResetForTesting()
	t.Cleanup(keyring.ResetForTesting)

	configHome := t.TempDir()
	setEnvCredentialTest(t, "XDG_CONFIG_HOME", configHome)
	if err := os.WriteFile(filepath.Join(configHome, "bk.yaml"), []byte("organizations:\n  my-org:\n    api_token: legacy-token\n"), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	conf := New(nil, nil)
	if got := conf.APITokenForOrg("my-org"); got != "legacy-token" {
		t.Fatalf("APITokenForOrg() = %q, want %q", got, "legacy-token")
	}
}

func TestConfig_APITokenForOrgPrefersEnvironmentToken(t *testing.T) {
	setEnvCredentialTest(t, "BUILDKITE_API_TOKEN", "env-token")
	setEnvCredentialTest(t, "BUILDKITE_NO_KEYRING", "")
	setEnvCredentialTest(t, "CI", "")
	setEnvCredentialTest(t, "BUILDKITE", "")

	conf := New(nil, nil)
	if got := conf.APITokenForOrg("ignored-org"); got != "env-token" {
		t.Fatalf("APITokenForOrg() = %q, want %q", got, "env-token")
	}
}

func TestConfig_RefreshTokenForOrgFromKeyring(t *testing.T) {
	keyring.MockForTesting()
	defer keyring.ResetForTesting()

	setEnvCredentialTest(t, "BUILDKITE_NO_KEYRING", "")
	setEnvCredentialTest(t, "CI", "")
	setEnvCredentialTest(t, "BUILDKITE", "")
	setEnvCredentialTest(t, "BUILDKITE_API_TOKEN", "")

	kr := keyring.New()
	if err := kr.SetRefreshToken("test-org", "refresh-token"); err != nil {
		t.Fatalf("SetRefreshToken() error = %v", err)
	}

	conf := New(nil, nil)
	if got := conf.RefreshTokenForOrg("test-org"); got != "refresh-token" {
		t.Fatalf("RefreshTokenForOrg() = %q, want %q", got, "refresh-token")
	}
}

func TestConfig_HasStoredTokenForOrgIncludesLegacyFallback(t *testing.T) {
	setEnvCredentialTest(t, "BUILDKITE_NO_KEYRING", "1")
	setEnvCredentialTest(t, "CI", "")
	setEnvCredentialTest(t, "BUILDKITE", "")
	setEnvCredentialTest(t, "BUILDKITE_API_TOKEN", "")
	keyring.ResetForTesting()
	t.Cleanup(keyring.ResetForTesting)

	configHome := t.TempDir()
	setEnvCredentialTest(t, "XDG_CONFIG_HOME", configHome)
	if err := os.WriteFile(filepath.Join(configHome, "bk.yaml"), []byte("organizations:\n  my-org:\n    api_token: legacy-token\n"), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	conf := New(nil, nil)
	if !conf.HasStoredTokenForOrg("my-org") {
		t.Fatal("expected HasStoredTokenForOrg to be true for legacy token")
	}
	if conf.HasStoredTokenForOrg("unknown-org") {
		t.Fatal("expected HasStoredTokenForOrg to be false for missing org")
	}
}

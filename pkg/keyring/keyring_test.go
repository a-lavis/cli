package keyring

import (
	"os"
	"path/filepath"
	"testing"
)

// setEnv sets an environment variable for the duration of the test and
// restores the original value (or unsets it) via t.Cleanup.
func setEnv(t *testing.T, key, value string) {
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
		// Reset the once so the next test starts fresh.
		ResetForTesting()
	})
	// Reset now so this test sees the new env value.
	ResetForTesting()
}

func writeUserConfig(t *testing.T, content string) {
	t.Helper()

	configHome := t.TempDir()
	setEnv(t, "XDG_CONFIG_HOME", configHome)

	if content == "" {
		return
	}

	if err := os.WriteFile(filepath.Join(configHome, "bk.yaml"), []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write user config: %v", err)
	}
}

func TestIsKeyringAvailable(t *testing.T) {
	// These tests manipulate package-level state (sync.Once) so must not run
	// in parallel with each other.

	t.Run("disabled by BUILDKITE_NO_KEYRING", func(t *testing.T) {
		writeUserConfig(t, "allow_keyring_in_ci: true\n")
		setEnv(t, "BUILDKITE_NO_KEYRING", "1")
		setEnv(t, "CI", "true")
		setEnv(t, "BUILDKITE", "")

		kr := New()
		if kr.IsAvailable() {
			t.Error("expected keyring to be unavailable when BUILDKITE_NO_KEYRING is set")
		}
	})

	t.Run("disabled by CI by default", func(t *testing.T) {
		writeUserConfig(t, "")
		setEnv(t, "CI", "true")
		setEnv(t, "BUILDKITE_NO_KEYRING", "")
		setEnv(t, "BUILDKITE", "")

		kr := New()
		if kr.IsAvailable() {
			t.Error("expected keyring to be unavailable when CI is set and allow_keyring_in_ci is not enabled")
		}
	})

	t.Run("enabled by CI opt-in from user config", func(t *testing.T) {
		writeUserConfig(t, "allow_keyring_in_ci: true\n")
		setEnv(t, "CI", "true")
		setEnv(t, "BUILDKITE_NO_KEYRING", "")
		setEnv(t, "BUILDKITE", "")

		kr := New()
		if !kr.IsAvailable() {
			t.Error("expected keyring to be available when CI is set and allow_keyring_in_ci is enabled")
		}
	})

	t.Run("disabled by BUILDKITE even with CI opt-in", func(t *testing.T) {
		writeUserConfig(t, "allow_keyring_in_ci: true\n")
		setEnv(t, "BUILDKITE", "true")
		setEnv(t, "BUILDKITE_NO_KEYRING", "")
		setEnv(t, "CI", "true")

		kr := New()
		if kr.IsAvailable() {
			t.Error("expected keyring to be unavailable when BUILDKITE is set")
		}
	})
}

func TestNoKeyringGet(t *testing.T) {
	setEnv(t, "BUILDKITE_NO_KEYRING", "1")
	setEnv(t, "CI", "")
	setEnv(t, "BUILDKITE", "")

	kr := New()
	token, err := kr.Get("my-org")
	if token != "" {
		t.Errorf("Get() returned non-empty token with keyring disabled, got %q", token)
	}
	if err == nil {
		t.Error("Get() expected ErrNotFound when keyring is disabled, got nil")
	}
}

func TestNoKeyringSet(t *testing.T) {
	setEnv(t, "BUILDKITE_NO_KEYRING", "1")
	setEnv(t, "CI", "")
	setEnv(t, "BUILDKITE", "")

	kr := New()
	if err := kr.Set("my-org", "token-123"); err != nil {
		t.Errorf("Set() returned unexpected error with keyring disabled: %v", err)
	}
}

func TestNoKeyringDelete(t *testing.T) {
	setEnv(t, "BUILDKITE_NO_KEYRING", "1")
	setEnv(t, "CI", "")
	setEnv(t, "BUILDKITE", "")

	kr := New()
	if err := kr.Delete("my-org"); err != nil {
		t.Errorf("Delete() returned unexpected error with keyring disabled: %v", err)
	}
}

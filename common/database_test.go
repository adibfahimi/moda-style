package common

import (
	"os"
	"testing"
)

// clearEnv unsets key for the duration of the test and restores its original
// value (or lack of one) afterwards. It is the inverse of t.Setenv and is used
// to exercise the defaults applied by GetDatabaseConfig.
func clearEnv(t *testing.T, key string) {
	t.Helper()

	original, hadValue := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unsetting %s failed: %v", key, err)
	}

	t.Cleanup(func() {
		if hadValue {
			os.Setenv(key, original)
			return
		}
		os.Unsetenv(key)
	})
}

func TestGetDatabaseConfig_ReadsEnvironmentVariables(t *testing.T) {
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("DB_PORT", "6543")
	t.Setenv("DB_USER", "moda")
	t.Setenv("DB_PASSWORD", "s3cret")
	t.Setenv("DB_NAME", "moda_style")
	t.Setenv("DB_SSL_MODE", "require")

	want := DatabaseConfig{
		Host:     "db.internal",
		Port:     "6543",
		User:     "moda",
		Password: "s3cret",
		DBName:   "moda_style",
		SSLMode:  "require",
	}

	if got := GetDatabaseConfig(); got != want {
		t.Errorf("GetDatabaseConfig() = %+v, want %+v", got, want)
	}
}

func TestGetDatabaseConfig_AppliesOptionalDefaults(t *testing.T) {
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("DB_USER", "moda")
	t.Setenv("DB_PASSWORD", "s3cret")
	t.Setenv("DB_NAME", "moda_style")
	clearEnv(t, "DB_PORT")
	clearEnv(t, "DB_SSL_MODE")

	config := GetDatabaseConfig()

	if config.Port != "5432" {
		t.Errorf("Port = %q, want the default %q", config.Port, "5432")
	}
	if config.SSLMode != "disable" {
		t.Errorf("SSLMode = %q, want the default %q", config.SSLMode, "disable")
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	const key = "MODA_TEST_OPTIONAL"

	t.Run("returns the configured value", func(t *testing.T) {
		t.Setenv(key, "configured")

		if got := getEnvOrDefault(key, "fallback"); got != "configured" {
			t.Errorf("getEnvOrDefault() = %q, want %q", got, "configured")
		}
	})

	t.Run("falls back when unset", func(t *testing.T) {
		clearEnv(t, key)

		if got := getEnvOrDefault(key, "fallback"); got != "fallback" {
			t.Errorf("getEnvOrDefault() = %q, want %q", got, "fallback")
		}
	})

	t.Run("falls back when empty", func(t *testing.T) {
		t.Setenv(key, "")

		if got := getEnvOrDefault(key, "fallback"); got != "fallback" {
			t.Errorf("getEnvOrDefault() = %q, want %q", got, "fallback")
		}
	})
}

func TestGetEnvOrFatal_ReturnsConfiguredValue(t *testing.T) {
	t.Setenv("MODA_TEST_REQUIRED", "present")

	if got := getEnvOrFatal("MODA_TEST_REQUIRED"); got != "present" {
		t.Errorf("getEnvOrFatal() = %q, want %q", got, "present")
	}
}

// ConnectDatabase and MustConnectDatabase are intentionally not unit tested:
// both require a reachable PostgreSQL server (and MustConnectDatabase terminates
// the process on failure). They are exercised by docker-compose integration
// runs instead.

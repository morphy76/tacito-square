package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultValues(t *testing.T) {
	v, err := Load("TS_TEST")
	require.NoError(t, err)

	assert.Equal(t, "info", v.GetString("log.level"))
	assert.Equal(t, "json", v.GetString("log.format"))
}

func TestLoad_EnvironmentOverride(t *testing.T) {
	os.Setenv("TS_TEST_LOG_LEVEL", "debug")
	defer os.Unsetenv("TS_TEST_LOG_LEVEL")

	v, err := Load("TS_TEST")
	require.NoError(t, err)

	assert.Equal(t, "debug", v.GetString("log.level"))
}

func TestLoadFromFile_InvalidPath(t *testing.T) {
	_, err := LoadFromFile("TS_TEST", "/nonexistent/config.yaml")
	assert.Error(t, err)
}

func TestLoad_CustomEnvironmentBindings(t *testing.T) {
	os.Setenv("TS_KEEPER_DB_URL", "postgres://test-url")
	os.Setenv("TS_KEEPER_CA_CERT_PATH", "/etc/ssl/ca.crt")
	defer func() {
		os.Unsetenv("TS_KEEPER_DB_URL")
		os.Unsetenv("TS_KEEPER_CA_CERT_PATH")
	}()

	v, err := Load("TS_KEEPER")
	require.NoError(t, err)

	assert.Equal(t, "postgres://test-url", v.GetString("database.url"))
	assert.Equal(t, "/etc/ssl/ca.crt", v.GetString("tls.caCertPath"))
}


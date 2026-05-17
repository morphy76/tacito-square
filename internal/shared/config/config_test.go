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

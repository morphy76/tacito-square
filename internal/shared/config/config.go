// Package config provides configuration loading for Tacito Square components
// using environment variables and config files via Viper.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)


// Load initializes Viper with defaults and environment variable binding.
// Prefix is the env var prefix (e.g., "TS_AGENT", "TS_KEEPER").
func Load(prefix string) (*viper.Viper, error) {
	v := viper.New()

	v.SetEnvPrefix(prefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// Handle database URL environment variable fallbacks
	dbURLEnv := prefix + "_DATABASE_URL"
	if os.Getenv(prefix+"_DB_URL") != "" {
		dbURLEnv = prefix + "_DB_URL"
	}
	_ = v.BindEnv("database.url", dbURLEnv)

	// Handle TLS CA certificate path environment variable fallbacks
	caCertEnv := prefix + "_TLS_CA_CERT_PATH"
	if os.Getenv(prefix+"_CA_CERT_PATH") != "" {
		caCertEnv = prefix + "_CA_CERT_PATH"
	}
	_ = v.BindEnv("tls.caCertPath", caCertEnv)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")

	return v, nil
}


// LoadFromFile loads configuration from a file path, falling back to env vars.
func LoadFromFile(prefix, path string) (*viper.Viper, error) {
	v, err := Load(prefix)
	if err != nil {
		return nil, err
	}

	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("reading config file %s: %w", path, err)
		}
	}

	return v, nil
}

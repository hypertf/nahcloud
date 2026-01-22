package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version is set at build time
var Version = "dev"

// Config holds all server configuration
type Config struct {
	Addr      string `mapstructure:"addr"`
	SQLiteDSN string `mapstructure:"sqlite_dsn"`
}

// setupConfig initializes viper with flags, env vars, and config file support
func setupConfig(cmd *cobra.Command) {
	// Define flags
	cmd.Flags().StringP("config", "c", "", "Config file path (YAML, JSON, or TOML)")
	cmd.Flags().String("addr", ":8080", "HTTP server address")
	cmd.Flags().String("sqlite-dsn", "", "SQLite database path")

	// Bind flags to viper
	viper.BindPFlag("addr", cmd.Flags().Lookup("addr"))
	viper.BindPFlag("sqlite_dsn", cmd.Flags().Lookup("sqlite-dsn"))

	// Set up environment variable binding with NAH_ prefix
	viper.SetEnvPrefix("NAH")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("addr", ":8080")
}

// loadConfig loads configuration from flags, env vars, and config file
func loadConfig(cmd *cobra.Command) (*Config, error) {
	// Check for config file
	configFile, _ := cmd.Flags().GetString("config")
	if configFile != "" {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// printConfigHelp prints additional help about environment variables
func printConfigHelp() string {
	return `
Environment Variables:
  All flags can be set via environment variables with the NAH_ prefix.
  Nested keys use underscores. Examples:

  NAH_ADDR=:9090                    Set server address
  NAH_SQLITE_DSN=./data.db          Set database path

Config File:
  Use --config to specify a YAML, JSON, or TOML config file.
  Example YAML config:

    addr: ":8080"
    sqlite_dsn: "./nahcloud.db"

Priority (highest to lowest):
  1. Command-line flags
  2. Environment variables
  3. Config file
  4. Defaults
`
}

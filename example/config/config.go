package config

import (
	"fmt"

	"github.com/vporoshok/envcfg"
)

// Config of the service
type Config struct {
	HTTPBind         string `default:":3000"`
	PasswordHashCost int    `default:"10"`
}

// MakeConfig create config and read it from environment
func MakeConfig() (*Config, error) {
	cfg := new(Config)
	err := envcfg.Read(cfg, envcfg.WithDefault(nil))
	return cfg, fmt.Errorf("read config: %w", err)
}

// GetPasswordHashCost is a getter
func (cfg *Config) GetPasswordHashCost() int {
	return cfg.PasswordHashCost
}

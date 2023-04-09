package config

import (
	"os"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

// Config stores the configuration for the backup program.
type Config struct {
	SourceDirs        []string `yaml:"source_dirs"`
	OutputDir         string   `yaml:"output_dir"`
	EncryptionKey     string   `yaml:"encryption_key"`
	RetentionDays     int      `yaml:"retention_days"`
	Interval          int      `yaml:"interval"`
	DiscordWebhookURL string   `yaml:"discord"`
}

// LoadConfig loads the backup configuration from a YAML file.
func LoadConfig(filePath string) (*Config, error) {
	// Read config file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parse YAML data
	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	// Check retention is positive
	if config.RetentionDays < 0 {
		log.Error().Msg("retention_days must be a positive number, check your config file")
		os.Exit(1)
	}

	// Check interval is positive
	if config.Interval < 0 {
		log.Error().Msg("interval must be a positive number, check your config file")
		os.Exit(1)
	}

	return config, nil
}

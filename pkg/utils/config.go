package utils

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type BackupConfig struct {
	Backups map[string]Backup `yaml:"backups"`
}

type Backup struct {
	Type      string    `yaml:"type"`
	Folder    []string  `yaml:"folder"`
	S3        S3config  `yaml:"s3"`
	Mysql     *Mysql    `yaml:"mysql,omitempty"`
	Mongo     *Mongo    `yaml:"mongo,omitempty"`
	Path      Path      `yaml:"path"`
	Retention Retention `yaml:"retention,omitempty"`
	Schedule  Schedule  `yaml:"schedule"`
}

type Mongo struct {
	Databases []string `yaml:"databases,omitempty"`
	Host      string   `yaml:"host,omitempty"`
	Port      string   `yaml:"port,omitempty"`
	User      string   `yaml:"user,omitempty"`
	Password  string   `yaml:"password,omitempty"`
	SSL       bool     `yaml:"ssl,omitempty"`
}

type Mysql struct {
	Databases []string `yaml:"databases,omitempty"`
	Host      string   `yaml:"host,omitempty"`
	Port      string   `yaml:"port,omitempty"`
	User      string   `yaml:"user,omitempty"`
	Password  string   `yaml:"password,omitempty"`
	SSL       string   `yaml:"ssl,omitempty"`
}

type Retention struct {
	Standard RetentionConfig `yaml:"standard"`
	Glacier  RetentionConfig `yaml:"glacier"`
}

type RetentionConfig struct {
	Days int `yaml:"days"`
}

type Schedule struct {
	Standard string `yaml:"standard"`
	Glacier  string `yaml:"glacier"`
}

type Path struct {
	Local string `yaml:"local"`
	S3    string `yaml:"s3"`
}

type S3config struct {
	Bucket     []string `yaml:"bucket"`
	Endpoint   string   `yaml:"endpoint"`
	Region     string   `yaml:"region"`
	ACCESS_KEY string   `yaml:"ACCESS_KEY"`
	SECRET_KEY string   `yaml:"SECRET_KEY"`
}

// LoadConfig loads configuration from a YAML file.
func LoadConfig(filepath string) (*BackupConfig, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config BackupConfig
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	return &config, nil
}

func GetConfig() (*BackupConfig, error) {
	configPath := GetEnv[string]("BACKUPS_CONFIG_PATH")
	// fmt.Printf("CONFIG_PATH: %s\n", configPath)
	if configPath == "" {
		if GetEnv[string]("GO_ENV") == "dev" {
			logger.Info("No config path provided, using default config path", source_utils)
			configPath = "config/config.yaml"
		}
		if GetEnv[string]("GO_ENV") == "prod" {
			logger.Info("No config path provided, using default config /etc/backup-tool/config.yaml", source_utils)
			configPath = "/etc/backup-tool/config.yaml"
		}
	}
	logger.Info(fmt.Sprintf("Loading config from %s", configPath), source_utils)
	config, err := LoadConfig(configPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load config: %v", err), source_utils)
		return nil, fmt.Errorf("failed to load config: %v", err)
	}
	return config, nil
}

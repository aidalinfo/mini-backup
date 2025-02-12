package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type BackupConfig struct {
	Backups map[string]Backup `yaml:"backups"`
}

type Backup struct {
	Type       string      `yaml:"type"`
	Folder     []string    `yaml:"folder"`
	S3         S3config    `yaml:"s3"`
	Mysql      *Mysql      `yaml:"mysql,omitempty"`
	Mongo      *Mongo      `yaml:"mongo,omitempty"`
	Sqlite     *Sqlite     `yaml:"sqlite,omitempty"`
	Kubernetes *Kubernetes `yaml:"kubernetes,omitempty"`
	Path       Path        `yaml:"path"`
	Retention  Retention   `yaml:"retention,omitempty"`
	Schedule   Schedule    `yaml:"schedule"`
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
	All bool `yaml:"all,omitempty"`
	Databases []string `yaml:"databases,omitempty"`
	Host      string   `yaml:"host,omitempty"`
	Port      string   `yaml:"port,omitempty"`
	User      string   `yaml:"user,omitempty"`
	Password  string   `yaml:"password,omitempty"`
	SSL       string   `yaml:"ssl,omitempty"`
}

type Kubernetes struct {
	KubeConfig string  `yaml:"kubeconfig"`
	Cluster    Cluster `yaml:"cluster"`
	Volumes    Volumes `yaml:"volumes"`
}

type Cluster struct {
	Backup   string   `yaml:"backup"`
	Excludes []string `yaml:"excludes"`
}

type Volumes struct {
	Enabled    bool     `yaml:"enabled"`
	AutoDetect bool     `yaml:"autodetect"`
	Excludes   []string `yaml:"excludes"`
	BackupPath string   `yaml:"backupPath"`
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
	All        bool     `yaml:"all"`
	Bucket     []string `yaml:"bucket"`
	Endpoint   string   `yaml:"endpoint"`
	PathStyle bool `yaml:"pathStyle"`
	Region     string   `yaml:"region"`
	ACCESS_KEY string   `yaml:"ACCESS_KEY"`
	SECRET_KEY string   `yaml:"SECRET_KEY"`
}

type Sqlite struct {
	Paths []string `yaml:"paths"`
	credentials struct {
		user     string `yaml:"user"`
		password string `yaml:"password"`
	}
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
	var configDir string
	if GetEnv[string]("GO_ENV") == "prod" {
		configDir = "/etc/backup-tool"
	} else {
		configDir = "config"
	}

	// Créer un BackupConfig vide pour stocker la configuration fusionnée pour renvoyer qu'un seul objet
	mergedConfig := &BackupConfig{
		Backups: make(map[string]Backup),
	}

	// Lire tous les fichiers du répertoire de configuration
	files, err := os.ReadDir(configDir)
	if err != nil {
		getLogger().Error(fmt.Sprintf("Failed to read config directory: %v", err), source_utils)
		return nil, fmt.Errorf("failed to read config directory: %v", err)
	}

	// Charger d'abord le fichier config.yaml principal s'il existe
	mainConfigPath := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(mainConfigPath); err == nil {
		getLogger().Info(fmt.Sprintf("Loading main config from %s", mainConfigPath), source_utils)
		mainConfig, err := LoadConfig(mainConfigPath)
		if err != nil {
			getLogger().Error(fmt.Sprintf("Failed to load main config: %v", err), source_utils)
		} else {
			for name, backup := range mainConfig.Backups {
				mergedConfig.Backups[name] = backup
			}
		}
	}

	// Parcourir chaque fichier pour trouver les .backups.yaml/.backups.yml afin de créer un objet BackupConfig
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".backups.yaml") || strings.HasSuffix(file.Name(), ".backups.yml")) {
			configPath := filepath.Join(configDir, file.Name())
			getLogger().Info(fmt.Sprintf("Loading backup config from %s", configPath), source_utils)

			config, err := LoadConfig(configPath)
			if err != nil {
				getLogger().Error(fmt.Sprintf("Failed to load config from %s: %v", configPath, err), source_utils)
				continue
			}

			// Fusionner les configurations
			for name, backup := range config.Backups {
				if _, exists := mergedConfig.Backups[name]; exists {
					getLogger().Error(fmt.Sprintf("Backup configuration '%s' from %s overrides existing configuration", name, configPath), source_utils)
				}
				mergedConfig.Backups[name] = backup
			}
		}
	}

	if len(mergedConfig.Backups) == 0 {
		return nil, fmt.Errorf("no valid backup configurations found in %s", configDir)
	}

	return mergedConfig, nil
}

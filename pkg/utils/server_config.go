package utils

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// ServerConfig contient la structure typée de la configuration.
type ServerConfig struct {
	Server        ServerSettings            `yaml:"server"`
	SecretManager map[string]SecretManager  `yaml:"secret_manager"`
	RStorage      map[string]RStorageConfig `yaml:"rstorage"`
	Modules       []string                  `yaml:"modules"`
}

type ServerSettings struct {
	Env   string `yaml:"env"`
	Port  string `yaml:"port"`
	Debug bool   `yaml:"debug"`
	Log   string `yaml:"log"`
}

type SecretManager struct {
	Name      string `yaml:"name"`
	URL       string `yaml:"url"`
	APIKey    string `yaml:"api_key"`
	ProjectID string `yaml:"project_id"`
}

type RStorageConfig struct {
	Endpoint   string `yaml:"endpoint"`
	BucketName string `yaml:"bucket_name"`
	PathStyle  bool   `yaml:"pathStyle"`
	AccessKey  string `yaml:"access_key"`
	SecretKey  string `yaml:"secret_key"`
	Region     string `yaml:"region"`
}

func GetConfigServer() (*ServerConfig, error) {
	// Définir le chemin par défaut
	configPath := os.Getenv("SERVER_CONFIG_PATH")
	if configPath == "" {
		configPath = "config/server.yaml"
	}
	// logger.Info(fmt.Sprintf("Loading config file: %s", configPath), source_utils)
	// Charger le fichier YAML
	file, err := os.Open(configPath)
	if err != nil {
		// logger.Error(fmt.Sprintf("Failed to open config file: %s", err), source_utils)
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config ServerConfig
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		// logger.Error(fmt.Sprintf("Failed to decode YAML: %s", err), source_utils)
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	// Résoudre les références aux variables d'environnement
	err = resolveEnvVariables(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve environment variables: %w", err)
	}

	return &config, nil
}

func resolveEnvVariables(config *ServerConfig) error {
	// Regex pour détecter les références comme ${{ENV_VAR}}
	envRegex := regexp.MustCompile(`\${{\s*(\w+)\s*}}`)
	resolve := func(value string) string {
		return envRegex.ReplaceAllStringFunc(value, func(match string) string {
			matches := envRegex.FindStringSubmatch(match)
			if len(matches) == 2 {
				if envValue := GetEnv[string](matches[1]); envValue != "" {
					return envValue
				}
			}
			return match
		})
	}

	// Résolution pour les paramètres du gestionnaire de secrets
	for key, sm := range config.SecretManager {
		sm.APIKey = resolve(sm.APIKey)
		sm.ProjectID = resolve(sm.ProjectID)
		config.SecretManager[key] = sm
	}

	// Résolution pour les configurations RStorage
	for key, storage := range config.RStorage {
		storage.AccessKey = resolve(storage.AccessKey)
		storage.SecretKey = resolve(storage.SecretKey)
		config.RStorage[key] = storage
	}

	// Résolution pour les paramètres du serveur
	config.Server.Env = resolve(config.Server.Env)
	config.Server.Port = resolve(config.Server.Port)
	config.Server.Log = resolve(config.Server.Log)
	return nil
}

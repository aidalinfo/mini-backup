package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type BackupConfig struct {
	Backups map[string]Backup `yaml:"backups"`
}

// Backup remplace les champs spécifiques par un champ générique.
type Backup struct {
	Type        string      `yaml:"type"`
	Folder      []string    `yaml:"folder"`
	GenericType any         `yaml:"-"` 
	Path        Path        `yaml:"path"`
	Retention   Retention   `yaml:"retention,omitempty"`
	Schedule    Schedule    `yaml:"schedule"`
}

// on décode en map, puis on extrait la config du type
func (b *Backup) UnmarshalYAML(node *yaml.Node) error {
	// On décode dans une map pour accéder à toutes les clés
	var raw map[string]interface{}
	if err := node.Decode(&raw); err != nil {
		return err
	}

	// Décodez ensuite les champs communs dans un alias pour éviter la récursion.
	type backupAlias Backup
	var alias backupAlias
	if err := node.Decode(&alias); err != nil {
		return err
	}
	*b = Backup(alias)

	// La configuration dynamique doit se trouver dans avoir la clé dont le nom correspond à la valeur de Type.
	if mod, ok := raw[b.Type]; ok {
		b.GenericType = mod
	} else {
		return fmt.Errorf("la configuration pour le type '%s' n'a pas été trouvée", b.Type)
	}

	return nil
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

	mergedConfig := &BackupConfig{
		Backups: make(map[string]Backup),
	}

	files, err := os.ReadDir(configDir)
	if err != nil {
		getLogger().Error(fmt.Sprintf("Failed to read config directory: %v", err), source_utils)
		return nil, fmt.Errorf("failed to read config directory: %v", err)
	}

	// Charger d'abord le fichier config.yaml principal s'il existe.
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

	// Parcourir chaque fichier pour trouver les .backups.yaml/.backups.yml
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

// BuildBackupArgs utilise GenericType pour construire les arguments de backup.
func BuildBackupArgs(backup Backup, glacierMode bool) (string, error) {
	result := make(map[string]interface{})

	// Champs communs
	result["path"] = backup.Path.Local
	result["Glaciermode"] = glacierMode

	result[strings.ToLower(backup.Type)] = backup.GenericType

	jsonData, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	getLogger().Info(fmt.Sprintf("JSON backup config: %s", string(jsonData)), source_utils)
	return string(jsonData), nil
}

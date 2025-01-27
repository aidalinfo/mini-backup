package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type RestoreConfig struct {
	Restores map[string]any `yaml:"restores"` // Map pour stocker les différents types de restauration
}

type KubernetesRestore struct {
	Type       string            `yaml:"type"`
	KubeConfig string            `yaml:"kubeconfig"`
	Cluster    KubernetesCluster `yaml:"cluster"`
	Volumes    KubernetesVolumes `yaml:"volumes"`
}

type KubernetesCluster struct {
	Provider   string      `yaml:"provider"`   // "same" ou "other"
	IDs        string      `yaml:"ids"`        // "same" ou "regenerate"
	Full       bool        `yaml:"full"`       // Restauration complète de l'état du cluster
	Namespaces []Namespace `yaml:"namespaces"` // Liste des namespaces
}

type KubernetesVolumes struct {
	Full       bool        `yaml:"full"`       // Restaurer tous les volumes
	Namespaces []Namespace `yaml:"namespaces"` // Liste des namespaces pour la restauration des volumes
	PVCs       []string    `yaml:"pvcs"`       // Liste des PVCs à restaurer
}

type Namespace struct {
	Name    string   `yaml:"name"`    // Nom du namespace
	Volumes []string `yaml:"volumes"` // Liste des volumes spécifiques à restaurer
}

// LoadRestoreConfig charge un fichier de configuration de restauration YAML unique.
func LoadRestoreConfig(filepath string) (*RestoreConfig, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open restore config file: %w", err)
	}
	defer file.Close()

	var config RestoreConfig
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode restore YAML: %w", err)
	}

	return &config, nil
}

// GetRestoreConfig charge et fusionne les configurations de restauration depuis plusieurs fichiers.
func GetRestoreConfig() (*RestoreConfig, error) {
	configDir := "config/restores"
	defaultConfigFile := filepath.Join(configDir, "restore_config.yaml")

	// Configuration fusionnée
	mergedConfig := &RestoreConfig{
		Restores: make(map[string]any),
	}

	// Charger le fichier de configuration par défaut s'il existe
	if _, err := os.Stat(defaultConfigFile); err == nil {
		defaultConfig, err := LoadRestoreConfig(defaultConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load default restore config: %w", err)
		}
		mergeRestoreConfigs(mergedConfig, defaultConfig)
	}

	// Charger tous les fichiers *.restores.yaml dans le répertoire
	files, err := os.ReadDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read restore config directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".restores.yaml") {
			configPath := filepath.Join(configDir, file.Name())
			config, err := LoadRestoreConfig(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load restore config from %s: %w", configPath, err)
			}
			mergeRestoreConfigs(mergedConfig, config)
		}
	}

	if len(mergedConfig.Restores) == 0 {
		return nil, fmt.Errorf("no valid restore configurations found in %s", configDir)
	}

	return mergedConfig, nil
}

// mergeRestoreConfigs fusionne deux configurations de restauration.
func mergeRestoreConfigs(base, toMerge *RestoreConfig) {
	for key, value := range toMerge.Restores {
		if _, exists := base.Restores[key]; exists {
			// Si une clé existe déjà, la configuration est ignorée (ou gérez les conflits si nécessaire)
			fmt.Printf("Warning: Restore configuration '%s' is overridden.\n", key)
		}
		base.Restores[key] = value
	}
}

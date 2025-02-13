package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type Module struct {
	Name    string `yaml:"name"`
	Bin     string `yaml:"bin"`
	Version string `yaml:"version"`
	Type    string `yaml:"type"`
	Enable  bool   `yaml:"enable"`
	Dir     string
}

var (
	ModulesMap = make(map[string]Module)
	Mu         sync.Mutex
)

func LoadModules() (map[string]Module, error) {
	modules := make(map[string]Module)
	modulesDir := "./modules"
	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read modules directory: %w", err)
	}

	fmt.Println("🔍 Chargement des modules...")

	for _, entry := range entries {
		if entry.IsDir() {
			modulePath := filepath.Join(modulesDir, entry.Name())
			yamlPath := filepath.Join(modulePath, "module.yaml")

			fmt.Println("📂 Vérification de :", yamlPath)

			if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
				fmt.Println("⚠️ Pas de module.yaml trouvé :", yamlPath)
				continue
			}

			data, err := os.ReadFile(yamlPath)
			if err != nil {
				fmt.Println("❌ Erreur de lecture de module.yaml :", yamlPath, err)
				continue
			}
			fmt.Println("📂 Contenu brut de module.yaml :", string(data))

			var genericMap map[string]Module
			if err := yaml.Unmarshal(data, &genericMap); err != nil {
				fmt.Println("❌ Erreur de parsing YAML :", yamlPath, err)
				continue
			}

			for key, mod := range genericMap {
				fmt.Println("🔍 Module détecté sous la clé :", key)
				mod.Dir = modulePath

				if mod.Enable {
					modules[mod.Type] = mod
					fmt.Printf("✅ Module chargé : %s (Type: %s, Version: %s) depuis %s\n", mod.Name, mod.Type, mod.Version, modulePath)
				}
				break // On prend le premier module trouvé
			}
		}
	}

	return modules, nil
}

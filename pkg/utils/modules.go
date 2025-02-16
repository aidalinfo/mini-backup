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

	fmt.Println("🔍 Chargement des modules...")

	err := filepath.WalkDir(modulesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Si on trouve un fichier nommé module.yaml
		if !d.IsDir() && d.Name() == "module.yaml" {
			fmt.Println("📂 Vérification de :", path)

			data, err := os.ReadFile(path)
			if err != nil {
				fmt.Println("❌ Erreur de lecture de module.yaml :", path, err)
				return nil
			}
			fmt.Println("📂 Contenu brut de module.yaml :", string(data))

			var genericMap map[string]Module
			if err := yaml.Unmarshal(data, &genericMap); err != nil {
				fmt.Println("❌ Erreur de parsing YAML :", path, err)
				return nil
			}

			// On considère que le dossier parent de module.yaml correspond au dossier du module
			moduleDir := filepath.Dir(path)

			for key, mod := range genericMap {
				fmt.Println("🔍 Module détecté sous la clé :", key)
				mod.Dir = moduleDir

				if mod.Enable {
					modules[mod.Type] = mod
					fmt.Printf("✅ Module chargé : %s (Type: %s, Version: %s) depuis %s\n", mod.Name, mod.Type, mod.Version, moduleDir)
				}
				break // on prend le premier module trouvé dans le fichier
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("erreur lors du parcours des modules: %w", err)
	}

	return modules, nil
}

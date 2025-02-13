package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Module struct {
	Bin     string `yaml:"bin"`
	Version string `yaml:"version"`
	Type    string `yaml:"type"`
	Enable  bool   `yaml:"enable"`
	Dir     string
}

var ModulesMap = make(map[string]Module)

func LoadModules() error {
	modulesDir := "./modules"
	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		return fmt.Errorf("failed to read modules directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			modulePath := filepath.Join(modulesDir, entry.Name())
			yamlPath := filepath.Join(modulePath, "module.yaml")
			if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
				getLogger().Info(fmt.Sprintf("No module.yaml found in %s, skipping", modulePath))
				continue
			}

			data, err := ioutil.ReadFile(yamlPath)
			if err != nil {
				getLogger().Error(fmt.Sprintf("Failed to read %s: %v", yamlPath, err))
				continue
			}

			var mod Module
			if err := yaml.Unmarshal(data, &mod); err != nil {
				getLogger().Error(fmt.Sprintf("Failed to parse YAML in %s: %v", yamlPath, err))
				continue
			}
			mod.Dir = modulePath
			if mod.Enable {
				ModulesMap[mod.Type] = mod
				getLogger().Info(fmt.Sprintf("Loaded module: %s (version %s) from %s", mod.Type, mod.Version, modulePath))
			}
		}
	}
	return nil
}

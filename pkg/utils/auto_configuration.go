package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	configDir  = "./config"
	envFile    = "./.env"
	serverYaml = "server.yaml"
	configYaml = "config.yaml"
	baseURL    = "https://raw.githubusercontent.com/aidalinfo/mini-backup/main/examples"
)

func AutoConfigurationFunc() error {
	// Vérifier et créer le dossier config si nécessaire
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0750); err != nil {
			return fmt.Errorf("erreur lors de la création du dossier config: %w", err)
		}
	}

	// Télécharger les fichiers YAML si nécessaires
	files := map[string]string{
		filepath.Join(configDir, serverYaml): baseURL + "/" + serverYaml,
		filepath.Join(configDir, configYaml): baseURL + "/" + configYaml,
	}

	for localPath, url := range files {
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			if err := downloadFile(localPath, url); err != nil {
				return fmt.Errorf("erreur lors du téléchargement de %s: %w", localPath, err)
			}
		}
	}

	return nil
}

func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

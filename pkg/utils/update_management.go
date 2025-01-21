package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
)

var REPO_URL = "https://github.com/aidalinfo/mini-backup"

// downloadAndReplace remplace un binaire spécifique (serveur ou CLI)
func downloadAndReplace(latestVersion, component string) error {
	arch := runtime.GOARCH
	osName := runtime.GOOS
	downloadURL := fmt.Sprintf("%s/releases/download/v%s/mini-backup-%s_%s_%s", REPO_URL, latestVersion, component, osName, arch)
	targetPath := fmt.Sprintf("/etc/mini-backup/mini-backup-%s", component)

	logger.Info(fmt.Sprintf("Téléchargement de la nouvelle version (%s)...", downloadURL), "UPDATE")

	tmpFile, err := os.CreateTemp("", "mini-backup-"+component)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la création du fichier temporaire : %v", err), "UPDATE")
		return fmt.Errorf("erreur lors de la création du fichier temporaire: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	resp, err := http.Get(downloadURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors du téléchargement : %v", err), "UPDATE")
		return fmt.Errorf("erreur lors du téléchargement: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error(fmt.Sprintf("Erreur lors du téléchargement : status %d", resp.StatusCode), "UPDATE")
		return fmt.Errorf("erreur lors du téléchargement: status %d", resp.StatusCode)
	}

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de l'écriture du fichier temporaire : %v", err), "UPDATE")
		return fmt.Errorf("erreur lors de l'écriture du fichier temporaire: %v", err)
	}

	tmpFile.Close()
	if err := os.Chmod(tmpFile.Name(), 0750); err != nil {
		logger.Error(fmt.Sprintf("Erreur lors du chmod : %v", err), "UPDATE")
		return fmt.Errorf("erreur lors du chmod: %v", err)
	}

	logger.Info(fmt.Sprintf("Installation du binaire %s dans %s...", component, targetPath), "UPDATE")
	err = os.Rename(tmpFile.Name(), targetPath)
	if err != nil {
		cmd := exec.Command("sudo", "mv", tmpFile.Name(), targetPath)
		if err := cmd.Run(); err != nil {
			logger.Error(fmt.Sprintf("Erreur lors du remplacement du binaire avec sudo : %v", err), "UPDATE")
			return fmt.Errorf("erreur lors du remplacement du binaire: %v", err)
		}
	}

	logger.Info(fmt.Sprintf("Mise à jour réussie de %s vers la version %s !", component, latestVersion), "UPDATE")
	return nil
}

// CheckForUpdates vérifie la dernière version disponible sur GitHub
func CheckForUpdates(currentVersion string) (string, error) {
	logger.Info("Vérification des mises à jour...", "UPDATE")

	releasesURL := fmt.Sprintf("%s/releases/latest", REPO_URL)
	resp, err := http.Get(releasesURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la vérification des mises à jour : %v", err), "UPDATE")
		return "", fmt.Errorf("erreur lors de la vérification des mises à jour : %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error(fmt.Sprintf("Erreur lors de la récupération des informations (status %d)", resp.StatusCode), "UPDATE")
		return "", fmt.Errorf("erreur lors de la récupération des informations (status %d)", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la lecture des données JSON : %v", err), "UPDATE")
		return "", fmt.Errorf("erreur lors de la lecture des données JSON : %v", err)
	}

	latestVersion := release.TagName
	if latestVersion == currentVersion {
		logger.Info("Vous utilisez déjà la dernière version.", "UPDATE")
		return "", nil
	}

	logger.Info(fmt.Sprintf("Nouvelle version disponible : %s", latestVersion), "UPDATE")
	return latestVersion, nil
}

// UpdateServer met à jour le binaire serveur
func UpdateServer(latestVersion string) error {
	logger.Info("Mise à jour du serveur...", "UPDATE")
	return downloadAndReplace(latestVersion, "server")
}

// UpdateCLI met à jour le binaire CLI
func UpdateCLI(latestVersion string) error {
	logger.Info("Mise à jour de la CLI...", "UPDATE")
	return downloadAndReplace(latestVersion, "cli")
}

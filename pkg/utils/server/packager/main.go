package packager

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"mini-backup/pkg/utils"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	// URL de l'index des modules
	IndexURL = "https://raw.githubusercontent.com/aidalinfo/modules-minibackup/main/.index.json"
	// URL de base pour le téléchargement des modules
	BaseURL = "https://pkg.aidalinfo.fr/repository/minibackup-modules"
)

// Module représente la structure d'un module dans le JSON.
type Module struct {
	
	Version  string            `json:"version"`
	Type     string            `json:"type"`
	Path     string            `json:"path"`
	Metadata map[string]string `json:"metadata"`
}

// ModuleIndex représente l'index complet des modules organisé par catégorie.
type ModuleIndex map[string]map[string]Module

// FetchModuleIndex télécharge et décode l'index JSON des modules.
func FetchModuleIndex() (ModuleIndex, error) {
	resp, err := http.Get(IndexURL)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération de l'index: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("statut HTTP inattendu: %s", resp.Status)
	}

	var index ModuleIndex
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&index); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage du JSON: %v", err)
	}

	return index, nil
}

// ListModules récupère l'index puis parcourt celui-ci pour lister les modules
// et construire leur URL de téléchargement.
func ListModules() ([]struct {
	Category    string
	Name        string
	ModuleInfo  Module
	DownloadURL string
}, error) {
	index, err := FetchModuleIndex()
	if err != nil {
		return nil, err
	}

	var modulesList []struct {
		Category    string
		Name        string
		ModuleInfo  Module
		DownloadURL string
	}

	for category, modules := range index {
		for name, module := range modules {
			downloadURL := fmt.Sprintf("%s/%s/%s.zip", BaseURL, category, name)
			modulesList = append(modulesList, struct {
				Category    string
				Name        string
				ModuleInfo  Module
				DownloadURL string
			}{
				Category:    category,
				Name:        name,
				ModuleInfo:  module,
				DownloadURL: downloadURL,
			})
		}
	}

	return modulesList, nil
}

// DownloadModule télécharge le fichier depuis l'URL et le sauvegarde dans outputPath.
func DownloadModule(downloadURL, outputPath string) error {
	// Création du dossier de destination si nécessaire
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("erreur lors de la création du dossier %s: %v", dir, err)
	}

	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("erreur lors du téléchargement depuis %s: %v", downloadURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("statut HTTP inattendu pour %s: %s", downloadURL, resp.Status)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("erreur lors de la création du fichier %s: %v", outputPath, err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("erreur lors de l'écriture dans le fichier %s: %v", outputPath, err)
	}

	return nil
}

// UnzipModule décompresse le fichier zip situé à zipPath dans le dossier destDir.
func UnzipModule(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("erreur lors de l'ouverture du fichier zip: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// Sécurité : vérifier que le chemin de destination ne sort pas du dossier destDir
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("chemin illégal: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			// Créer le dossier
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		// Créer le dossier parent si nécessaire
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		inFile, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			inFile.Close()
			return err
		}

		_, err = io.Copy(outFile, inFile)
		inFile.Close()
		outFile.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func CheckModuleVersion(mod utils.Module) error {
	// Récupération de l'index distant
	index, err := FetchModuleIndex()
	if err != nil {
		return fmt.Errorf("erreur lors de la récupération de l'index distant: %v", err)
	}

	var remoteVersion string
	found := false

	for _, modules := range index {
		for key, remoteMod := range modules {
			if key == mod.Type || key == mod.Name {
				remoteVersion = remoteMod.Version
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return fmt.Errorf("module %s non trouvé dans l'index distant", mod.Name)
	}

	// Comparaison des versions
	if mod.Version == remoteVersion {
		fmt.Printf("✅ Le module %s (version %s) est à jour.\n", mod.Name, mod.Version)
	} else {
		fmt.Printf("⚠️ Mise à jour disponible pour le module %s : version locale = %s, version distante = %s.\n", mod.Name, mod.Version, remoteVersion)
	}

	return nil
}

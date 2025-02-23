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

var logger = utils.LoggerFunc()

const (
	IndexURL = "https://raw.githubusercontent.com/aidalinfo/modules-minibackup/main/.index.json"
	BaseURL = "https://pkg.aidalinfo.fr/repository/minibackup-modules"
)

type Module struct {
	Version  string            `json:"version"`
	Type     string            `json:"type"`
	Path     string            `json:"path"`
	Metadata map[string]string `json:"metadata"`
}

type ModuleIndex map[string]map[string]Module

func FetchModuleIndex() (ModuleIndex, error) {
	resp, err := http.Get(IndexURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la récupération de l'index : %v", err))
		return nil, fmt.Errorf("erreur lors de la récupération de l'index: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error(fmt.Sprintf("Statut HTTP inattendu: %s", resp.Status))
		return nil, fmt.Errorf("statut HTTP inattendu: %s", resp.Status)
	}

	var index ModuleIndex
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&index); err != nil {
		logger.Error(fmt.Sprintf("Erreur lors du décodage du JSON : %v", err))
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
		logger.Error(fmt.Sprintf("Erreur lors de la création du dossier %s : %v", dir, err))
		return fmt.Errorf("erreur lors de la création du dossier %s: %v", dir, err)
	}

	resp, err := http.Get(downloadURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors du téléchargement depuis %s : %v", downloadURL, err))
		return fmt.Errorf("erreur lors du téléchargement depuis %s: %v", downloadURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error(fmt.Sprintf("Statut HTTP inattendu pour %s: %s", downloadURL, resp.Status))
		return fmt.Errorf("statut HTTP inattendu pour %s: %s", downloadURL, resp.Status)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la création du fichier %s : %v", outputPath, err))
		return fmt.Errorf("erreur lors de la création du fichier %s: %v", outputPath, err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de l'écriture dans le fichier %s : %v", outputPath, err))
		return fmt.Errorf("erreur lors de l'écriture dans le fichier %s: %v", outputPath, err)
	}

	return nil
}

// UnzipModule décompresse le fichier zip situé à zipPath dans le dossier destDir.
func UnzipModule(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de l'ouverture du fichier zip : %v", err))
		return fmt.Errorf("erreur lors de l'ouverture du fichier zip: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// Sécurité :le chemin de destination ne sort pas du dossier destDir
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

// CheckModuleVersion compare la version locale d'un module avec celle de l'index distant.
func CheckModuleVersion(mod utils.Module) (bool, error) {
	// Récupération de l'index distant
	index, err := FetchModuleIndex()
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la récupération de l'index distant : %v", err))
		return false, fmt.Errorf("erreur lors de la récupération de l'index distant: %v", err)
	}

	var remoteVersion string
	found := false

	for _, modules := range index {
		for key, remoteMod := range modules {
			// On compare avec le nom du module 
			if key == mod.Name {
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
		logger.Error(fmt.Sprintf("Module %s non trouvé dans l'index distant", mod.Name))
		return false, fmt.Errorf("module %s non trouvé dans l'index distant", mod.Name)
	}

	// Comparaison des versions
	if mod.Version == remoteVersion {
		logger.Info(fmt.Sprintf("Le module %s (version %s) est à jour.", mod.Name, mod.Version))
		fmt.Printf("✅ Le module %s (version %s) est à jour.\n", mod.Name, mod.Version)
		return false, nil
	} else {
		logger.Info(fmt.Sprintf("Mise à jour disponible pour le module %s : version locale = %s, version distante = %s.", mod.Name, mod.Version, remoteVersion))
		fmt.Printf("⚠️ Mise à jour disponible pour le module %s : version locale = %s, version distante = %s.\n", mod.Name, mod.Version, remoteVersion)
		return true, nil
	}
}

type ModulePackage struct {
	Category    string
	Name        string
	ModuleInfo  Module
	DownloadURL string
}

func InstallModule(mod ModulePackage) error {
	// Chemin pour enregistrer le fichier zip téléchargé
	zipPath := filepath.Join("modules", mod.Category, mod.Name+".zip")
	logger.Info(fmt.Sprintf("Téléchargement du module %s...", mod.Name))
	if err := DownloadModule(mod.DownloadURL, zipPath); err != nil {
		logger.Error(fmt.Sprintf("Erreur lors du téléchargement du module %s : %v", mod.Name, err))
		return err
	}
	logger.Info(fmt.Sprintf("Module %s téléchargé avec succès", mod.Name))

	// Définir le dossier de destination pour la décompression
	unzipDest := filepath.Join("modules", mod.Category, mod.Name)
	logger.Info(fmt.Sprintf("Décompression du module %s...", mod.Name))
	if err := UnzipModule(zipPath, unzipDest); err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la décompression du module %s : %v", mod.Name, err))
		return err
	}
	logger.Info(fmt.Sprintf("Module %s décompressé avec succès", mod.Name))

	// Suppression du fichier zip téléchargé
	if err := os.Remove(zipPath); err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la suppression du fichier zip %s : %v", mod.Name, err))
		return err
	}

	return nil
}
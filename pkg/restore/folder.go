package restore

import (
	"fmt"
	"io"
	"mini-backup/pkg/utils"
	"os"
	"path/filepath"
)

// RestoreFolder restaure un ou plusieurs fichiers/dossiers depuis un chemin donné vers `config.Folder[0]`.
func RestoreFolder(restorePath string, config utils.Backup) error {
	logger := utils.LoggerFunc()

	// Assurez-vous que config.Folder contient au moins une destination valide
	if len(config.Folder) == 0 || config.Folder[0] == "" {
		return fmt.Errorf("aucun chemin de destination défini dans config.Folder")
	}

	destination := config.Folder[0]

	logger.Info(fmt.Sprintf("Starting folder restore from %s to %s", restorePath, destination))

	// Vérifier si le dossier de destination existe, sinon le créer
	err := os.MkdirAll(destination, 0755)
	if err != nil {
		return fmt.Errorf("erreur lors de la création du dossier de destination %s : %v", destination, err)
	}

	// Restaurer le contenu de `restorePath` dans `destination`
	info, err := os.Stat(restorePath)
	if err != nil {
		return fmt.Errorf("erreur lors de l'accès au chemin source %s : %v", restorePath, err)
	}

	if info.IsDir() {
		// Restaurer tout le contenu d'un dossier source
		err = restoreFolderContents(restorePath, destination)
		if err != nil {
			return fmt.Errorf("erreur lors de la restauration du dossier %s vers %s : %v", restorePath, destination, err)
		}
		logger.Info(fmt.Sprintf("Dossier %s restauré avec succès vers %s", restorePath, destination))
	} else {
		// Restaurer un fichier individuel
		destPath := filepath.Join(destination, filepath.Base(restorePath))
		err = copyFile(restorePath, destPath)
		if err != nil {
			return fmt.Errorf("erreur lors de la restauration du fichier %s vers %s : %v", restorePath, destPath, err)
		}
		logger.Info(fmt.Sprintf("Fichier %s restauré avec succès vers %s", restorePath, destPath))
	}

	return nil
}

// restoreFolderContents copie récursivement le contenu d'un dossier source vers un dossier destination.
func restoreFolderContents(srcPath, destPath string) error {
	err := filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("erreur lors de l'accès au chemin %s : %v", path, err)
		}

		// Calculer le chemin destination relatif
		relativePath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return fmt.Errorf("erreur lors du calcul du chemin relatif pour %s : %v", path, err)
		}
		targetPath := filepath.Join(destPath, relativePath)

		if info.IsDir() {
			// Créer les sous-dossiers
			err = os.MkdirAll(targetPath, 0755)
			if err != nil {
				return fmt.Errorf("erreur lors de la création du dossier %s : %v", targetPath, err)
			}
		} else {
			// Copier les fichiers
			err = copyFile(path, targetPath)
			if err != nil {
				return fmt.Errorf("erreur lors de la copie du fichier %s vers %s : %v", path, targetPath, err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("erreur lors de la restauration du dossier %s : %v", srcPath, err)
	}

	return nil
}

// copyFile copie un fichier d'une source vers une destination.
func copyFile(srcPath, destPath string) error {
	// Ouvrir le fichier source
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("erreur lors de l'ouverture du fichier source %s : %v", srcPath, err)
	}
	defer srcFile.Close()

	// Créer le fichier de destination
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("erreur lors de la création du fichier destination %s : %v", destPath, err)
	}
	defer destFile.Close()

	// Copier le contenu du fichier
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("erreur lors de la copie de %s vers %s : %v", srcPath, destPath, err)
	}

	logger.Debug(fmt.Sprintf("Fichier %s copié avec succès vers %s", srcPath, destPath))
	return nil
}

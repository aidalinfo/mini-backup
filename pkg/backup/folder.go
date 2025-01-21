package backup

import (
	"fmt"
	"mini-backup/pkg/utils"
	"os"
	"path/filepath"
	"time"
)

// CopyFolder copie un ou plusieurs dossiers et leur contenu vers un dossier de destination.
func CopyFolder(name string, config utils.Backup) ([]string, error) {
	logger := utils.LoggerFunc()

	paths := config.Folder
	destination := config.Path.Local
	date := time.Now().Format("20060102_150405")

	// Vérifier si le dossier de destination existe, sinon le créer
	err := os.MkdirAll(destination, 0755)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la création du dossier de destination %s : %v", destination, err)
	}

	var foldersCopied []string

	// Parcourir les chemins donnés
	for _, srcPath := range paths {
		baseName := filepath.Base(srcPath)
		newFolderName := fmt.Sprintf("%s-%s-%s", name, baseName, date)
		newDestination := filepath.Join(destination, newFolderName)

		logger.Info(fmt.Sprintf("Copie du contenu de %s vers %s", srcPath, newDestination))
		err := utils.CopyDir(newDestination, srcPath)
		if err != nil {
			logger.Error(fmt.Sprintf("Erreur lors de la copie de %s : %v", srcPath, err))
			continue
		}

		foldersCopied = append(foldersCopied, newDestination)
		logger.Info(fmt.Sprintf("Dossier %s copié avec succès vers %s", srcPath, newDestination))
	}

	return foldersCopied, nil
}

package restore

import (
	"fmt"
	"mini-backup/pkg/utils"
	"os"
	"path/filepath"
	"strings"
)

var logger = utils.LoggerFunc()

// CoreRestore gère la logique de restauration
func CoreRestore(name string, version string) error {
	logger.Info(fmt.Sprintf("Starting restore process for: %s, version: %s", name, version))
	config, err := utils.GetConfig()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load config: %v", err))
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Error(fmt.Sprintf("Panic occurred during restore for %s: %v", name, r))
		}
	}()

	backupConfig, ok := config.Backups[name]
	if !ok {
		err := fmt.Errorf("no backup configuration found for: %s", name)
		logger.Error(err.Error())
		return err
	}

	switch backupConfig.Type {
	case "mysql":
		logger.Info(fmt.Sprintf("Detected MySQL restore for %s", name))
		result, err := restoreProcess(name, backupConfig, version)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to restore MySQL for %s: %v", name, err))
			return err
		}
		return RestoreMySQL(result, backupConfig)
	case "folder":
		logger.Info(fmt.Sprintf("Detected folder restore for %s", name))
		result, err := restoreProcess(name, backupConfig, version)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to restore folder for %s: %v", name, err))
			return err
		}
		return RestoreFolder(result, backupConfig)
	// case "s3":
	// 	logger.Info(fmt.Sprintf("Detected S3 restore for %s", name))
	// 	return RestoreS3(name, version, backupConfig)
	case "mongo":
		logger.Info(fmt.Sprintf("Detected MongoDB restore for %s", name))
		result, err := restoreProcess(name, backupConfig, version)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to restore MongoDB for %s: %v", name, err))
			return err
		}
		fmt.Print("Resultat de la restauration: ", result)
		return RestoreMongoDB(result, backupConfig)
	default:
		err := fmt.Errorf("unsupported restore type: %s", backupConfig.Type)
		logger.Error(err.Error())
		return err
	}
}

// restoreProcess gère le téléchargement, le déchiffrement et la décompression d'un fichier de sauvegarde.
func restoreProcess(name string, config utils.Backup, version string) (string, error) {
	logger := utils.LoggerFunc()
	logger.Info(fmt.Sprintf("Starting restore process for: %s, version: %s", name, version))

	// Charger la configuration du serveur
	configServer, err := utils.GetConfigServer()
	if err != nil {
		logger.Error("Failed to load server configuration")
		return "", err
	}

	// Vérifier si RStorage contient au moins un élément
	if len(configServer.RStorage) == 0 {
		err := fmt.Errorf("no storage configuration found")
		logger.Error(err.Error())
		return "", err
	}

	// Obtenir le premier élément de RStorage
	var firstStorageName string
	var firstStorageConfig utils.RStorageConfig
	for name, config := range configServer.RStorage {
		firstStorageName = name
		firstStorageConfig = config
		break
	}

	// Initialiser le client S3 avec le premier storage
	s3client, err := utils.RstorageManager(firstStorageName, &firstStorageConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to initialize S3 manager: %v", err))
		return "", err
	}

	var targetFile string

	if version == "last" {
		// Télécharger le dernier fichier depuis S3
		logger.Info(fmt.Sprintf("Searching for latest backup in: %s", config.Path.S3))
		files, err := s3client.ListBackups(config.Path.S3)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to list backups in S3 path: %v", err))
			return "", err
		}

		// Trouver le dernier fichier
		for _, file := range files {
			if strings.HasPrefix(filepath.Base(file), name) && strings.HasSuffix(file, ".enc") {
				if targetFile == "" || file > targetFile {
					targetFile = file
				}
			}
		}

		if targetFile == "" {
			err := fmt.Errorf("no backup file found for %s in %s", name, config.Path.S3)
			logger.Error(err.Error())
			return "", err
		}

		logger.Info(fmt.Sprintf("Found latest backup: %s", targetFile))
	} else {
		// Utiliser la version spécifiée
		logger.Info(fmt.Sprintf("Using specified backup version: %s", version))
		targetFile = filepath.Join(config.Path.S3, version)
	}

	// Télécharger le fichier chiffré
	localEncryptedPath := filepath.Join(config.Path.Local, filepath.Base(targetFile))
	err = s3client.Download(targetFile, localEncryptedPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to download %s: %v", targetFile, err))
		return "", err
	}
	logger.Info(fmt.Sprintf("Downloaded encrypted file to: %s", localEncryptedPath))

	// Déchiffrer le fichier
	localDecryptedPath := strings.TrimSuffix(localEncryptedPath, ".enc")
	err = utils.DecryptFile(localEncryptedPath, localDecryptedPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to decrypt file %s: %v", localEncryptedPath, err))
		return "", err
	}
	logger.Info(fmt.Sprintf("Decrypted file to: %s", localDecryptedPath))

	// Supprimer le fichier chiffré local
	err = deleteFile(localEncryptedPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to delete encrypted file %s: %v", localEncryptedPath, err))
	}

	// Décompresser le fichier si nécessaire
	var finalPath string
	if strings.HasSuffix(localDecryptedPath, ".gz") && config.Type != "mongo" {
		finalPath = strings.TrimSuffix(localDecryptedPath, ".gz")
		output, err := utils.Decompress(localDecryptedPath, finalPath)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to decompress file %s: %v", localDecryptedPath, err))
			return "", err
		}
		logger.Info(fmt.Sprintf("Decompressed file to: %s", output))
		finalPath = output
		deleteFile(localDecryptedPath)
	} else {
		finalPath = localDecryptedPath
		logger.Info(fmt.Sprintf("No decompression needed for: %s", finalPath))
	}

	return finalPath, nil
}

func deleteFile(path string) error {
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression du fichier %s : %v", path, err)
	}
	return nil
}

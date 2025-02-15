package restore

import (
	"encoding/json"
	"fmt"
	"mini-backup/pkg/utils"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var logger = utils.LoggerFunc()

type ModuleOutput struct {
	Logs  map[string][]string `json:"logs"`
	State bool                `json:"state"`
}

func CoreRestore(name string, backupFile string, restoreName string, restoreParams any) error {
	logger.Info(fmt.Sprintf("Starting restore for: %s", name))

	config, err := utils.GetConfig()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load config: %v", err))
		return err
	}

	modules, err := utils.LoadModules()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load modules: %v", err))
		return err
	}
	logger.Debug(fmt.Sprintf("📌 Modules chargés : %v", modules))

	backupConfig, ok := config.Backups[name]
	if !ok {
		err := fmt.Errorf("no backup configuration found for: %s", name)
		logger.Error(err.Error())
		return err
	}

	mod, ok := modules[backupConfig.Type]
	if !ok {
		err := fmt.Errorf("unsupported restore type: %s", backupConfig.Type)
		logger.Error(err.Error())
		return err
	}

	restoreArgs, err := utils.BuildBackupArgs(config.Backups[name], false)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la création du JSON restoreArgs : %v", err))
		return err
	}
	backupPath, err := restoreProcess(name, backupConfig, backupFile)
	if err != nil {
		logger.Error(fmt.Sprintf("restoreProcess error: %v", err))
		return err
	}
	binPath := filepath.Join(mod.Dir, mod.Bin)
	cmd := exec.Command(binPath, "restore", name, backupPath, restoreArgs)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to restore %s for %s: %v, output: %s", backupConfig.Type, name, err, string(output)))
		return err
	}

	logger.Info(fmt.Sprintf("Output of restore command: %s", string(output)))

	var moduleOutput ModuleOutput
	err = json.Unmarshal(output, &moduleOutput)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors du parsing du JSON de sortie: %v", err))
		return err
	}

	for level, logs := range moduleOutput.Logs {
		for _, msg := range logs {
			logger.Info(fmt.Sprintf("[%s] %s", level, msg), fmt.Sprintf("module_restore_%s", name))
		}
	}

	if !moduleOutput.State {
		logger.Error("❌ Aucun chemin de restauration trouvé dans la sortie JSON")
		return fmt.Errorf("no restore path found")
	}
	logger.Info(fmt.Sprintf("Successfully restored %s for %s", backupConfig.Type, name))
	return nil
}

// // mapToStruct convertit une map[string]interface{} en une structure donnée.
// func mapToStruct(input map[string]interface{}, output interface{}) error {
// 	jsonData, err := json.Marshal(input)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal map to JSON: %w", err)
// 	}
// 	if err := json.Unmarshal(jsonData, output); err != nil {
// 		return fmt.Errorf("failed to unmarshal JSON to struct: %w", err)
// 	}
// 	return nil
// }

// restoreProcess gère le téléchargement, le déchiffrement et la décompression d'un fichier de sauvegarde.
func restoreProcess(name string, config utils.Backup, backupFile string) (string, error) {
	logger := utils.LoggerFunc()
	logger.Info(fmt.Sprintf("Starting restore process for: %s, backupFile: %s", name, backupFile), "[RESTORE] [CORE]")

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
		logger.Error(fmt.Sprintf("Failed to initialize S3 manager: %v", err), "[RESTORE] [CORE]")
		return "", err
	}

	var targetFile string

	if backupFile == "last" {
		// Télécharger le dernier fichier depuis S3
		logger.Info(fmt.Sprintf("Searching for latest backup in: %s", config.Path.S3), "[RESTORE] [CORE]")
		files, err := s3client.ListBackups(config.Path.S3)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to list backups in S3 path: %v", err), "[RESTORE] [CORE]")
			return "", err
		}
		logger.Debug(fmt.Sprintf("Found files: %v", files))

		// Trouver le dernier fichier
		for _, file := range files {
			if strings.HasSuffix(file, ".enc") {
				if targetFile == "" || file > targetFile {
					targetFile = file
				}
			}
		}

		if targetFile == "" {
			err := fmt.Errorf("no backup file found for %s in %s", name, config.Path.S3)
			logger.Error(err.Error(), "[RESTORE] [CORE]")
			return "", err
		}

		logger.Info(fmt.Sprintf("Found latest backup: %s", targetFile))
	} else {
		// Utiliser la backupFile spécifiée
		logger.Info(fmt.Sprintf("Using specified backup backupFile: %s", backupFile))
		targetFile = backupFile
	}

	// Télécharger le fichier chiffré
	localEncryptedPath := filepath.Join(config.Path.Local, filepath.Base(targetFile))
	err = s3client.Download(targetFile, localEncryptedPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to download %s: %v", targetFile, err), "[RESTORE] [CORE]")
		return "", err
	}
	logger.Info(fmt.Sprintf("Downloaded encrypted file to: %s", localEncryptedPath), "[RESTORE] [CORE]")

	// Déchiffrer le fichier
	localDecryptedPath := strings.TrimSuffix(localEncryptedPath, ".enc")
	err = utils.DecryptFile(localEncryptedPath, localDecryptedPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to decrypt file %s: %v", localEncryptedPath, err), "[RESTORE] [CORE]")
		return "", err
	}
	logger.Info(fmt.Sprintf("Decrypted file to: %s", localDecryptedPath), "[RESTORE] [CORE]")

	// Supprimer le fichier chiffré local
	err = deleteFile(localEncryptedPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to delete encrypted file %s: %v", localEncryptedPath, err), "[RESTORE] [CORE]")
	}

	// Décompresser le fichier si nécessaire
	var finalPath string
	if strings.HasSuffix(localDecryptedPath, ".gz") && config.Type != "mongo" {
		// Dans le cas type file ou s3
		if strings.HasSuffix(localDecryptedPath, ".tar.gz") {
			finalPath = strings.TrimSuffix(localDecryptedPath, ".tar.gz")
		} else {
			finalPath = strings.TrimSuffix(localDecryptedPath, ".gz")
		}

		output, err := utils.Decompress(localDecryptedPath, finalPath)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to decompress file %s: %v", localDecryptedPath, err), "[RESTORE] [CORE]")
			return "", err
		}
		logger.Info(fmt.Sprintf("Decompressed file to: %s", output), "[RESTORE] [CORE]")
		finalPath = output
		deleteFile(localDecryptedPath)
	} else {
		finalPath = localDecryptedPath
		logger.Info(fmt.Sprintf("No decompression needed for: %s", finalPath), "[RESTORE] [CORE]")
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

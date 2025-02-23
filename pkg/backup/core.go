package backup

import (
	"bufio"
	"fmt"
	"mini-backup/pkg/utils"
	"os"
	"os/exec"
	"path/filepath"
)

var logger = utils.LoggerFunc()

func backupProcess(path []string, config utils.Backup, backupName string, glacierMode bool) error {
	compressedPath := []string{}
	for _, p := range path {
		var compressed string
		if filepath.Ext(p) == ".gz" {
			logger.Info(fmt.Sprintf("File %s is already compressed, skipping compression.", path))
			compressed = p
			compressedPath = append(compressedPath, p)
		} else {
			// Compresser le fichier
			cp, err := utils.Compress(p)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to compress %s: %v", path, err))
				return err
			}
			logger.Info(fmt.Sprintf("Successfully compressed %s", path))
			compressed = cp
			compressedPath = append(compressedPath, cp)
		}
		encryptedPath := compressed + ".enc"
		utils.EncryptFile(compressed, encryptedPath)
		logger.Info(fmt.Sprintf("Successfully compressed %s", p))
		logger.Debug(fmt.Sprintf("Compressed paths: %v", compressedPath))
		configServer, err := utils.GetConfigServer()
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to get config server: %v", err))
			return err
		}
		for name, configServer := range configServer.RStorage {
			s3client, err := utils.RstorageManager(name, &configServer)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to get storage manager: %v", err))
				continue
			}
			s3client.ManageRetention(filepath.Join(config.Path.S3, filepath.Base(encryptedPath)), config.Retention.Standard.Days, glacierMode)
			s3FilePath := filepath.Join(config.Path.S3, filepath.Base(encryptedPath))
			err = s3client.Upload(encryptedPath, s3FilePath, glacierMode)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to upload %s to %s: %v", encryptedPath, configServer.BucketName, err))
				continue
			}
			logger.Info(fmt.Sprintf("Successfully uploaded %s to %s", encryptedPath, configServer.BucketName))
		}
		deleteFile(p)
		deleteFile(compressed)
		deleteFile(encryptedPath)
	}
	fmt.Println(compressedPath)
	logger.Info(fmt.Sprintf("[TRACING] : Backup OK : %s ", backupName), "BACKUP PROCESS")
	return nil
}

func deleteFile(path string) error {
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression du fichier %s : %v", path, err)
	}
	return nil
}

func CoreBackup(name string, glacierMode bool) error {
	logger.Info(fmt.Sprintf("Starting backup for: %s", name))
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
	fmt.Println("📌 Modules chargés :", modules) // DEBUG

	backupType := config.Backups[name].Type
	mod, ok := modules[backupType]
	if !ok {
		err := fmt.Errorf("unsupported backup type: %s", backupType)
		logger.Error(err.Error())
		return err
	}

	// Création du JSON des arguments de backup à partir du config.yaml
	backupArgs, err := utils.BuildBackupArgs(config.Backups[name], glacierMode)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la création du JSON backupArgs : %v", err))
		return err
	}
	binPath := filepath.Join(mod.Dir, mod.Bin)
	cmd := exec.Command(binPath, "backup", name, backupArgs)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
			logger.Error(fmt.Sprintf("Erreur lors de la création du stdout pipe: %v", err))
			return err
	}
	var lastLine string
	go func() {
			scanner := bufio.NewScanner(stdoutPipe)
			for scanner.Scan() {
					logger.Info(scanner.Text(), fmt.Sprintf("[MODULE] [%s]", mod.Name))
					line := scanner.Text()
					lastLine = line                           
			}
			if err := scanner.Err(); err != nil {
					logger.Error(fmt.Sprintf("Erreur lors de la lecture de stdout: %v", err))
			}
	}()

	// Démarrer la commande
	if err := cmd.Start(); err != nil {
			logger.Error(fmt.Sprintf("Erreur lors du démarrage de la commande: %v", err))
			return err
	}
	// Attendre la fin de l'exécution de la commande
	if err := cmd.Wait(); err != nil {
			logger.Error(fmt.Sprintf("Erreur lors de l'exécution de la commande: %v", err))
			return err
	}
	backupPath := string(lastLine)

	logger.Info(fmt.Sprintf("Backup path extrait: %s", backupPath))

	backupProcess([]string{backupPath}, config.Backups[name], name, glacierMode)
	logger.Info(fmt.Sprintf("Successfully backed up %s for %s", backupType, name))
	return nil
}

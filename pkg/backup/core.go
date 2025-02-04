package backup

import (
	"fmt"
	"mini-backup/pkg/utils"
	"os"
	"path/filepath"
)

var logger = utils.LoggerFunc()

func backupProcess(path []string, config utils.Backup) error {
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
				continue
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
			continue
		}
		for name, configServer := range configServer.RStorage {
			s3client, err := utils.RstorageManager(name, &configServer)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to get storage manager: %v", err))
				continue
			}
			s3client.ManageRetention(filepath.Join(config.Path.S3, filepath.Base(encryptedPath)), config.Retention.Standard.Days)
			s3FilePath := filepath.Join(config.Path.S3, filepath.Base(encryptedPath))
			err = s3client.Upload(encryptedPath, s3FilePath)
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
	return nil
}

func deleteFile(path string) error {
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression du fichier %s : %v", path, err)
	}
	return nil
}

func CoreBackup(name string) error {
	logger.Info(fmt.Sprintf("Starting backup for: %s", name))
	config, err := utils.GetConfig()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load config: %v", err))
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Error(fmt.Sprintf("Panic occurred during backup for %s: %v", name, r))
		}
	}()

	switch config.Backups[name].Type {
	case "mysql":
		logger.Info(fmt.Sprintf("Detected MySQL backup for %s", name))
		result, err := BackupMySQL(name, config.Backups[name])
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to backup MySQL for %s: %v", name, err))
			return err
		}
		logger.Info(fmt.Sprintf("Successfully backed up MySQL for %s: %v", name, result))
		backupProcess(result, config.Backups[name])
		return nil
	case "folder":
		logger.Info(fmt.Sprintf("Detected folder backup for %s", name))
		result, err := CopyFolder(name, config.Backups[name])
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to backup folder for %s: %v", name, err))
			return err
		}
		logger.Debug(fmt.Sprintln("Resultat de la copie de dossier:", result))
		backupProcess(result, config.Backups[name])
		logger.Info(fmt.Sprintf("Successfully backed up folder for %s", name))
		return nil
	case "s3":
		logger.Info(fmt.Sprintf("Detected S3 backup for %s", name))
		result, err := BackupRemoteS3(name, config.Backups[name])
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to backup S3 for %s: %v", name, err))
			return err
		}
		backupProcess(result, config.Backups[name])
		logger.Info(fmt.Sprintf("Successfully backed up S3 for %s: %v", name, result))
		return nil
	case "mongo":
		logger.Info(fmt.Sprintf("Detected MongoDB backup for %s", name))
		result, err := BackupMongoDB(name, config.Backups[name])
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to backup MongoDB for %s: %v", name, err))
			return err
		}
		resultArray := []string{result}
		backupProcess(resultArray, config.Backups[name])
		logger.Info(fmt.Sprintf("Successfully backed up MongoDB for %s: %v", name, result))
		return nil
	case "sqlite":
		logger.Info(fmt.Sprintf("Detected SQLite backup for %s", name))
		result, err := BackupSqlite(name, config.Backups[name])
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to backup SQLite for %s: %v", name, err))
			return err
		}
		resultArray := []string{result}
		backupProcess(resultArray, config.Backups[name])
		logger.Info(fmt.Sprintf("Successfully backed up SQLite for %s: %v", name, result))
		return nil
	case "kubernetes":
		logger.Info(fmt.Sprintf("Detected Kubernetes backup for %s", name))
		result, err := BackupKube(name, config.Backups[name])
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to backup Kubernetes for %s: %v", name, err))
			return err
		}
		backupProcess(result, config.Backups[name])
		logger.Info(fmt.Sprintf("Successfully backed up Kubernetes for %s", name))
		return nil
	default:
		err := fmt.Errorf("unsupported backup type: %s", config.Backups[name].Type)
		logger.Error(err.Error())
		return err
	}
}

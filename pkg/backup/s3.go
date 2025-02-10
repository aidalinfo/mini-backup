package backup

import (
	"fmt"
	"mini-backup/pkg/utils"
	"path/filepath"
	"time"
)

func BackupRemoteS3(name string, config utils.Backup) ([]string, error) {
	logger.Debug(fmt.Sprintf("Information de connexion S3 : %v", config.S3))

	// Création du fichier credentials AWS
	err := utils.AwsCredentialFileCreateFunc(config.S3.ACCESS_KEY, config.S3.SECRET_KEY, name)
	if err != nil {
		return nil, err
	}

	// Formatage du dossier parent avec timestamp
	date := time.Now().Format("20060102_150405")
	parentDir := fmt.Sprintf("%s/%s_s3_backup_%s", config.Path.Local, name, date)

	// Liste des buckets à sauvegarder
	var bucketsToBackup []string

	// Si `All` est activé, lister tous les buckets disponibles
	if config.S3.All {
		s3client, err := utils.NewS3Manager("", config.S3.Region, config.S3.Endpoint, name, config.S3.PathStyle)
		if err != nil {
			logger.Error(fmt.Sprintf("Erreur lors de l'initialisation du gestionnaire S3 : %v", err))
			return nil, err
		}

		buckets, err := s3client.ListBuckets()
		if err != nil {
			logger.Error(fmt.Sprintf("Erreur lors de la récupération de la liste des buckets S3 : %v", err))
			return nil, err
		}

		bucketsToBackup = buckets
	} else {
		bucketsToBackup = config.S3.Bucket
	}

	// Liste des chemins de backup
	allBucketPath := []string{}

	// Sauvegarde chaque bucket
	for _, bucket := range bucketsToBackup {
		logger.Debug(fmt.Sprintf("Backup du bucket S3 : %s", bucket))

		s3client, err := utils.NewS3Manager(bucket, config.S3.Region, config.S3.Endpoint, name, config.S3.PathStyle)
		if err != nil {
			logger.Error(fmt.Sprintf("Erreur lors de l'initialisation du gestionnaire S3 pour %s : %v", bucket, err))
			continue
		}

		// Destination du backup local
		destinationPath := filepath.Join(parentDir, fmt.Sprintf("%s-%s-%s", name, bucket, date))

		// Copier le backup depuis S3
		err = s3client.CopyBackupToLocal(destinationPath)
		if err != nil {
			logger.Error(fmt.Sprintf("Erreur lors de la copie du backup depuis S3 pour %s : %v", bucket, err))
			continue
		}

		allBucketPath = append(allBucketPath, destinationPath)
	}

	logger.Info(fmt.Sprintf("Backup copié avec succès depuis les buckets S3 : %v", allBucketPath))
	return []string{parentDir}, nil
}


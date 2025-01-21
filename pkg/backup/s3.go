package backup

import (
	"fmt"
	"mini-backup/pkg/utils"
	"path/filepath"
	"time"
)

func BackupRemoteS3(name string, config utils.Backup) ([]string, error) {
	credentialHeader := "[" + name + "]"
	logger.Debug(fmt.Sprintf("Information de connexion S3 : %v", config.S3))
	err := utils.AwsCredentialFileCreateFunc(config.S3.ACCESS_KEY, config.S3.SECRET_KEY, credentialHeader)
	if err != nil {
		return nil, err
	}
	allBucketPath := []string{}
	for _, bucket := range config.S3.Bucket {
		logger.Debug(fmt.Sprintf("Backup du bucket S3 : %s", bucket))
		s3client, err := utils.NewS3Manager(bucket, config.S3.Region, config.S3.Endpoint, name)
		if err != nil {
			logger.Error(fmt.Sprintf("Erreur lors de l'initialisation du gestionnaire S3 : %v\n", err))
			continue
		}
		destinationPath := filepath.Join(config.Path.Local, fmt.Sprintf("%s-%s-%s", name, bucket, time.Now().Format("20060102_150405")))
		// Copier le backup depuis S3
		errcopy := s3client.CopyBackupToLocal(destinationPath)
		if errcopy != nil {
			logger.Error(fmt.Sprintf("Erreur lors de la copie du backup depuis S3 : %v\n", errcopy))
			continue
		}
		allBucketPath = append(allBucketPath, destinationPath)
	}
	logger.Info(fmt.Sprintf("Backup copié avec succès depuis les buckets S3 : %v", allBucketPath))
	return allBucketPath, nil
}

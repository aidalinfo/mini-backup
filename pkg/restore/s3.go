package restore

import (
	"fmt"
	"mini-backup/pkg/utils"
	"os"
	"path/filepath"
	"strings"
)

func RestoreS3(backupPath string, config utils.Backup, name string) error {
    logger.Info(fmt.Sprintf("Starting S3 restore process from: %s", backupPath), "[RESTORE] [S3]")

    err := utils.AwsCredentialFileCreateFunc(config.S3.ACCESS_KEY, config.S3.SECRET_KEY, name)
    if err != nil {
        logger.Error(fmt.Sprintf("Erreur lors de la génération du fichier AWS credentials : %v", err), "[RESTORE] [S3]")
        return err
    }

    // Lister les dossiers dans backupPath (chaque dossier représente un bucket)
    entries, err := os.ReadDir(backupPath)
    if err != nil {
        logger.Error(fmt.Sprintf("Erreur lors de la lecture du dossier de backup %s : %v", backupPath, err), "[RESTORE] [S3]")
        return err
    }

    // Restaurer chaque bucket individuellement
    for _, entry := range entries {
        if !entry.IsDir() {
            continue // Ignorer les fichiers à la racine
        }

        fullBucketPath := entry.Name() // ex: "name-bucket-data-20250210_150405"

        // Trouver le dernier `-` avant la date pour bien extraire le `bucketName`
        lastDash := strings.LastIndex(fullBucketPath, "-")
        if lastDash == -1 {
            logger.Error(fmt.Sprintf("Nom de dossier invalide : %s, ignoré.", fullBucketPath), "[RESTORE] [S3]")
            continue
        }

        bucketName := fullBucketPath[len(name)+1 : lastDash] // Extraction correcte du `bucketName`
        bucketPath := filepath.Join(backupPath, fullBucketPath)

        logger.Info(fmt.Sprintf("Restoring bucket: %s from %s", bucketName, bucketPath), "[RESTORE] [S3]")

        // Initialiser le gestionnaire S3 pour ce bucket
        s3client, err := utils.NewS3Manager(bucketName, config.S3.Region, config.S3.Endpoint, name, config.S3.PathStyle)
        if err != nil {
            logger.Error(fmt.Sprintf("Erreur lors de l'initialisation du gestionnaire S3 pour %s : %v", bucketName, err), "[RESTORE] [S3]")
            continue
        }

        // Vérifier si le bucket existe et le créer si nécessaire
        exists, err := s3client.DoesBucketExist(bucketName)
        if err != nil {
            logger.Error(fmt.Sprintf("Erreur lors de la vérification du bucket %s : %v", bucketName, err), "[RESTORE] [S3]")
            continue
        }
        if !exists {
            err := s3client.CreateBucket(bucketName)
            if err != nil {
                logger.Error(fmt.Sprintf("Erreur lors de la création du bucket %s : %v", bucketName, err), "[RESTORE] [S3]")
                continue
            }
            logger.Info(fmt.Sprintf("Bucket %s créé avec succès", bucketName), "[RESTORE] [S3]")
        }
        // Uploader les fichiers et dossiers du bucket
        err = filepath.Walk(bucketPath, func(path string, info os.FileInfo, err error) error {
            if err != nil {
                return err
            }

            // Calculer le chemin relatif
            relPath, err := filepath.Rel(bucketPath, path)
            if err != nil {
                return fmt.Errorf("failed to get relative path: %v", err)
            }

            // Vérifier que le chemin ne contient pas "." pour les ./fileName
            if relPath == "." {
                return nil // On ignore ce cas
            }

            if info.IsDir() {
                // Créer un dossier dans S3
                err = s3client.UploadEmptyFolder(relPath + "/")
                if err != nil {
                    logger.Error(fmt.Sprintf("Erreur lors de la création du dossier %s dans S3 : %v", relPath, err), "[RESTORE] [S3]")
                    return fmt.Errorf("failed to create folder %s: %v", relPath, err)
                }
                logger.Info(fmt.Sprintf("Successfully created folder: %s in bucket: %s", relPath, bucketName), "[RESTORE] [S3]")
                return nil
            }

            // Upload du fichier vers le bucket correspondant
            err = s3client.Upload(path, relPath)
            if err != nil {
                logger.Error(fmt.Sprintf("Erreur lors du téléversement du fichier %s : %v", path, err), "[RESTORE] [S3]")
                return fmt.Errorf("failed to upload file %s: %v", path, err)
            }

            logger.Info(fmt.Sprintf("Successfully uploaded file: %s to bucket: %s", relPath, bucketName), "[RESTORE] [S3]")
            return nil
        })

        if err != nil {
            logger.Error(fmt.Sprintf("Échec de la restauration pour le bucket %s : %v", bucketName, err), "[RESTORE] [S3]")
            continue
        }

        logger.Info(fmt.Sprintf("Successfully restored all files to S3 bucket: %s", bucketName), "[RESTORE] [S3]")
    }

    logger.Info("S3 restore process completed successfully.", "[RESTORE] [S3]")
    return nil
}


package restore

import (
	"fmt"
	"mini-backup/pkg/utils"
	"os"
	"path/filepath"
)

func RestoreS3(backupPath string, config utils.Backup, name string) error {
    logger.Info(fmt.Sprintf("Starting S3 restore process from: %s", backupPath), "[RESTORE] [S3]")
    
    err := utils.AwsCredentialFileCreateFunc(config.S3.ACCESS_KEY, config.S3.SECRET_KEY, name)
    if err != nil {
        logger.Error(fmt.Sprintf("Erreur lors de la génération du fichier AWS credentials : %v", err), "[RESTORE] [S3]")
        return err
    }

    // Initialiser le client S3
    s3client, err := utils.NewS3Manager(
        config.S3.Bucket[0],
        config.S3.Region,
        config.S3.Endpoint,
        name,
        config.S3.PathStyle,
    )
    if err != nil {
        logger.Error(fmt.Sprintf("Erreur lors de l'initialisation du gestionnaire S3 : %v\n", err), "[RESTORE] [S3]")
        return fmt.Errorf("failed to initialize S3 manager: %v", err)
    }

    // Parcourir récursivement le dossier et uploader chaque fichier
    err = filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        // Ignorer les dossiers
        if info.IsDir() {
            return nil
        }

        // Calculer le chemin relatif pour le préserver dans S3
        relPath, err := filepath.Rel(backupPath, path)
        if err != nil {
            return fmt.Errorf("failed to get relative path: %v", err)
        }

        // Upload le fichier
        err = s3client.Upload(path, relPath)
        if err != nil {
            logger.Error(fmt.Sprintf("Erreur lors du téléversement du fichier %s : %v", path, err), "[RESTORE] [S3]")
            return fmt.Errorf("failed to upload file %s: %v", path, err)
        }

        logger.Info(fmt.Sprintf("Successfully uploaded file: %s", relPath), "[RESTORE] [S3]")
        return nil
    })

    if err != nil {
        return fmt.Errorf("failed to restore to S3: %v", err)
    }

    logger.Info(fmt.Sprintf("Successfully restored all files to S3 bucket: %s", config.S3.Bucket[0]), "[RESTORE] [S3]")
    return nil
}
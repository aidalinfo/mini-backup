package backup

import (
	"fmt"
	"mini-backup/pkg/utils"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// BackupMongoDB sauvegarde une ou toutes les bases de données MongoDB.
func BackupMongoDB(name string, config utils.Backup) (string, error) {
	logger.Info(fmt.Sprintf("Starting MongoDB backup for: %s", name))

	// Construire le chemin de sauvegarde local
	destinationPath := filepath.Join(config.Path.Local, fmt.Sprintf("%s-%s.bson.gz", name, time.Now().Format("20060102_150405")))
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0755); err != nil {
		logger.Error(fmt.Sprintf("Failed to create backup directory: %v", err))
		return "", err
	}

	// Construire l'URI de connexion MongoDB
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%s",
		config.Mongo.User, config.Mongo.Password, config.Mongo.Host, config.Mongo.Port,
	)
	if config.Mongo.SSL {
		uri += "?ssl=true"
	}

	// Construire la commande mongodump
	cmdArgs := []string{
		"--uri", uri,
		"--gzip",                       // Activer la compression gzip
		"--archive=" + destinationPath, // Sauvegarder dans un fichier compressé
	}

	// Exécuter la commande
	cmd := exec.Command("mongodump", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(fmt.Sprintf("mongodump failed: %s", string(output)))
		return "", fmt.Errorf("mongodump failed: %w", err)
	}

	logger.Info(fmt.Sprintf("MongoDB backup saved to: %s", destinationPath))
	return destinationPath, nil
}

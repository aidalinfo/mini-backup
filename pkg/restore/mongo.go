package restore

import (
	"bytes"
	"fmt"
	"mini-backup/pkg/utils"
	"os"
	"os/exec"
	"path/filepath"
)

// RestoreMongoDB restaure une base de données MongoDB à partir d'un fichier .bson.gz.
func RestoreMongoDB(backupPath string, config utils.Backup) error {
	logger := utils.LoggerFunc()

	logger.Info(fmt.Sprintf("Starting MongoDB restore from: %s", backupPath))

	// Vérifier la configuration MongoDB
	if config.Mongo.Host == "" || config.Mongo.User == "" {
		return fmt.Errorf("invalid MongoDB configuration: missing required fields (Host: %s, User: %s)", config.Mongo.Host, config.Mongo.User)
	}

	// Vérifier si le fichier de sauvegarde existe
	info, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("backup path not found: %s, error: %v", backupPath, err)
	}

	if info.IsDir() {
		return fmt.Errorf("backup path must be a .bson.gz file, not a directory: %s", backupPath)
	}

	// Vérifier l'extension du fichier
	if filepath.Ext(backupPath) != ".gz" {
		return fmt.Errorf("unsupported backup file format: %s (expected .bson.gz)", backupPath)
	}

	// Construire la commande mongorestore
	cmdArgs := []string{
		"--gzip",
		"--uri", fmt.Sprintf("mongodb://%s:%s@%s:%s/",
			config.Mongo.User, config.Mongo.Password, config.Mongo.Host, config.Mongo.Port),
		"--archive=" + backupPath,
	}

	cmd := exec.Command("mongorestore", cmdArgs...)

	// Capturer la sortie standard et les erreurs
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Exécuter la commande
	err = cmd.Run()
	logger.Debug(fmt.Sprintf("mongorestore stdout: %s", stdout.String()))
	logger.Debug(fmt.Sprintf("mongorestore stderr: %s", stderr.String()))

	if err != nil {
		logger.Error(fmt.Sprintf("MongoDB restore failed: %s", stderr.String()))
		return fmt.Errorf("mongorestore failed: %v", err)
	}

	logger.Info(fmt.Sprintf("MongoDB restore completed successfully from: %s", backupPath))
	return nil
}

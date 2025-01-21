package restore

import (
	"fmt"
	"mini-backup/pkg/utils"
	"os"
	"os/exec"
	"strings"
)

// RestoreMySQL performs a MySQL restore for one or multiple databases.
func RestoreMySQL(backupFile string, config utils.Backup) error {
	logger := utils.LoggerFunc()

	logger.Info(fmt.Sprintf("Starting MySQL restore from file: %s", backupFile))

	// Vérifie que la configuration est valide
	if config.Mysql.Host == "" || config.Mysql.User == "" {
		return fmt.Errorf("invalid MySQL configuration: missing required fields (Host: %s, User: %s)", config.Mysql.Host, config.Mysql.User)
	}

	// Vérifie si le fichier de sauvegarde existe
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupFile)
	}

	// Restaurer chaque base de données
	for _, database := range config.Mysql.Databases {
		err := restoreFunc(backupFile, config, database, logger)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to restore database %s from file %s: %v", database, backupFile, err))
			return err
		}
		logger.Info(fmt.Sprintf("Successfully restored database: %s", database))
	}

	logger.Info(fmt.Sprintf("Successfully restored all databases from file: %s", backupFile))
	return nil
}

// restoreFunc executes the mysql command to restore a single database.
func restoreFunc(backupFile string, config utils.Backup, database string, logger *utils.Logger) error {
	logger.Info(fmt.Sprintf("Restoring database: %s from file: %s", database, backupFile))

	// Commande mysql
	cmd := exec.Command(
		"mysql",
		"-h", config.Mysql.Host,
		"-P", config.Mysql.Port,
		"-u", config.Mysql.User,
		fmt.Sprintf("-p%s", config.Mysql.Password),
		database, // Sélectionner la base de données spécifique
	)

	// Vérifie si le fichier contient des instructions USE DATABASE
	fileContent, err := os.ReadFile(backupFile)
	if err != nil {
		return fmt.Errorf("failed to read backup file %s: %w", backupFile, err)
	}
	if !strings.Contains(string(fileContent), "USE") {
		// Ajouter une instruction USE pour s'assurer que la base de données est sélectionnée
		tempFile := strings.Replace(backupFile, ".sql", fmt.Sprintf("-%s.sql", database), 1)
		logger.Info(fmt.Sprintf("Creating temporary file with database context: %s", tempFile))

		err = os.WriteFile(tempFile, append([]byte(fmt.Sprintf("USE `%s`;\n", database)), fileContent...), 0644)
		if err != nil {
			return fmt.Errorf("failed to create temporary backup file: %w", err)
		}
		backupFile = tempFile
		defer os.Remove(tempFile) // Nettoyer le fichier temporaire après restauration
	}

	// Rediriger l'entrée de la commande à partir du fichier de sauvegarde
	file, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file %s: %w", backupFile, err)
	}
	defer file.Close()

	cmd.Stdin = file

	// Exécuter la commande
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(fmt.Sprintf("MySQL restore failed: %s", string(output)))
		return fmt.Errorf("mysql restore failed: %w", err)
	}

	logger.Info(fmt.Sprintf("Restore completed successfully for database: %s from file: %s", database, backupFile))
	return nil
}

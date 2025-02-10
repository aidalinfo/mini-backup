package restore

import (
	"encoding/json"
	"fmt"
	"mini-backup/pkg/utils"
	"os"
	"os/exec"
	"path/filepath"
)

// RestoreMySQL restaure une ou plusieurs bases de données à partir d'un dossier de sauvegarde.
func RestoreMySQL(name string, config utils.Backup, backupDir string, params any) error {
	logger := utils.LoggerFunc()
	logger.Info(fmt.Sprintf("Starting MySQL restore from directory: %s", backupDir))

	// Vérifie la configuration MySQL
	if config.Mysql.Host == "" || config.Mysql.User == "" {
		return fmt.Errorf("invalid MySQL configuration: missing required fields (Host: %s, User: %s)", config.Mysql.Host, config.Mysql.User)
	}

	// Vérifie que le dossier de backup existe
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return fmt.Errorf("backup directory not found: %s", backupDir)
	}

	// Identifier le fichier "all_databases.sql" s'il existe
	allDatabasesFile := filepath.Join(backupDir, fmt.Sprintf("%s-all_databases.sql", name))
	hasAllDatabasesBackup := fileExists(allDatabasesFile)

	// Bases de données à restaurer
	var databasesToRestore []string

	// Lecture des paramètres JSON (s'il y en a)
	if params != nil && params != "" {
		jsonData, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to parse restore parameters: %w", err)
		}

		var requestData struct {
			Databases []string `json:"databases"`
		}
		if err := json.Unmarshal(jsonData, &requestData); err != nil {
			return fmt.Errorf("failed to decode JSON restore parameters: %w", err)
		}

		// Utilise les bases demandées via JSON
		if len(requestData.Databases) > 0 {
			databasesToRestore = requestData.Databases
		}
	}

	// Si aucun paramètre spécifique, utiliser la config
	if len(databasesToRestore) == 0 {
		if config.Mysql.AllDatabases {
			// Restaurer toute la BDD si le fichier "all_databases.sql" existe
			if hasAllDatabasesBackup {
				return restoreAllDatabases(allDatabasesFile, config, logger)
			}
			return fmt.Errorf("all_databases.sql file not found in %s", backupDir)
		} else {
			// Sinon, restaurer uniquement les bases listées dans la config
			databasesToRestore = config.Mysql.Databases
		}
	}

	// Restaurer chaque base de données spécifiée
	for _, database := range databasesToRestore {
		dbBackupFile := filepath.Join(backupDir, fmt.Sprintf("%s-%s.sql", name, database))

		if fileExists(dbBackupFile) {
			// Restaurer une base de données avec son propre fichier SQL
			err := restoreSingleDatabase(dbBackupFile, config, database, logger)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to restore database %s: %v", database, err))
				return err
			}
		} else if hasAllDatabasesBackup {
			// Si "all_databases.sql" est présent, restaurer uniquement la base demandée
			err := restoreDatabaseFromAllDatabases(allDatabasesFile, config, database, logger)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to restore database %s from all_databases.sql: %v", database, err))
				return err
			}
		} else {
			logger.Error(fmt.Sprintf("No backup found for database: %s", database))
		}
	}

	logger.Info("MySQL restore completed successfully.")
	return nil
}

// restoreAllDatabases restaure l'ensemble des bases de données
func restoreAllDatabases(backupFile string, config utils.Backup, logger *utils.Logger) error {
	logger.Info(fmt.Sprintf("Restoring all databases from backup file: %s", backupFile))

	cmd := exec.Command(
		"mysql",
		"-h", config.Mysql.Host,
		"-P", config.Mysql.Port,
		"-u", config.Mysql.User,
		fmt.Sprintf("-p%s", config.Mysql.Password),
		"--set-gtid-purged=OFF",  
	)

	file, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file %s: %w", backupFile, err)
	}
	defer file.Close()

	cmd.Stdin = file

	// Exécuter la restauration
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(fmt.Sprintf("MySQL restore failed: %s", string(output)))
		return fmt.Errorf("mysql restore failed: %w", err)
	}

	logger.Info("Successfully restored all databases.")
	return nil
}

// restoreDatabaseFromAllDatabases restaure une base spécifique à partir de all_databases.sql
func restoreDatabaseFromAllDatabases(backupFile string, config utils.Backup, database string, logger *utils.Logger) error {
	logger.Info(fmt.Sprintf("Restoring database %s from all_databases.sql", database))

	cmd := exec.Command(
		"mysql",
		"-h", config.Mysql.Host,
		"-P", config.Mysql.Port,
		"-u", config.Mysql.User,
		fmt.Sprintf("-p%s", config.Mysql.Password),
		"--one-database", database,
		"--set-gtid-purged=OFF",  
	)

	file, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("failed to open all_databases backup file %s: %w", backupFile, err)
	}
	defer file.Close()

	cmd.Stdin = file

	// Exécuter la restauration
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(fmt.Sprintf("MySQL restore failed for database %s: %s", database, string(output)))
		return fmt.Errorf("mysql restore failed: %w", err)
	}

	logger.Info(fmt.Sprintf("Successfully restored database %s from all_databases.sql", database))
	return nil
}

// restoreSingleDatabase restaure une seule base de données depuis un fichier SQL dédié
func restoreSingleDatabase(backupFile string, config utils.Backup, database string, logger *utils.Logger) error {
	logger.Info(fmt.Sprintf("Restoring database: %s from file: %s", database, backupFile))

	cmd := exec.Command(
		"mysql",
		"-h", config.Mysql.Host,
		"-P", config.Mysql.Port,
		"-u", config.Mysql.User,
		fmt.Sprintf("-p%s", config.Mysql.Password),
		database,
	)

	file, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file %s: %w", backupFile, err)
	}
	defer file.Close()

	cmd.Stdin = file

	// Exécuter la restauration
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(fmt.Sprintf("MySQL restore failed: %s", string(output)))
		return fmt.Errorf("mysql restore failed: %w", err)
	}

	logger.Info(fmt.Sprintf("Restore completed successfully for database: %s", database))
	return nil
}

// fileExists vérifie si un fichier existe
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

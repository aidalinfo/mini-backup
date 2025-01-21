package backup

import (
	"fmt"
	"mini-backup/pkg/utils"
	"os"
	"os/exec"
	"time"
)

// BackupMySQL performs a MySQL dump using the mysqldump command-line tool.
func BackupMySQL(name string, config utils.Backup) ([]string, error) {
	fmt.Println("BackupMySQL")
	fmt.Println(config)
	// Vérifie que la configuration est valide
	if config.Mysql.Host == "" || config.Mysql.Databases[0] == "" || config.Mysql.User == "" {
		return []string{}, fmt.Errorf("invalid MySQL configuration: missing required fields (Host: %s, Database: %s, User: %s)", config.Mysql.Host, config.Mysql.Databases[0], config.Mysql.User)
	}
	dumping := []string{}

	// Créer le répertoire de sauvegarde
	outputDir := config.Path.Local
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return []string{}, fmt.Errorf("failed to create backup directory: %w", err)
	}

	for _, database := range config.Mysql.Databases {
		result, err := dumpFunc(name, config, database, outputDir, logger)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to dump database %s: %v", database, err))
			continue
		}
		dumping = append(dumping, result)
		logger.Info(fmt.Sprintf("Successfully dumped database: %s", database))
	}

	return dumping, nil
}

// executes mysqldump to dump a single database.
func dumpFunc(name string, config utils.Backup, database, outputDir string, logger *utils.Logger) (string, error) {
	// Chemin du fichier de sauvegarde
	date := time.Now().Format("20060102_150405")
	filename := name + "-" + database + "-" + date
	outputFile := fmt.Sprintf("%s/%s.sql", outputDir, filename)

	// Commande mysqldump
	cmd := exec.Command(
		"mysqldump",
		"-h", config.Mysql.Host,
		"-P", config.Mysql.Port,
		"-u", config.Mysql.User,
		"--ssl="+config.Mysql.SSL,
		fmt.Sprintf("-p%s", config.Mysql.Password),
		database,
	)

	// Rediriger la sortie vers le fichier
	file, err := os.Create(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file for database %s: %w", database, err)
	}
	defer file.Close()

	cmd.Stdout = file

	// Exécuter la commande
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("mysqldump failed for database %s: %w", database, err)
	}

	logger.Info(fmt.Sprintf("Backup saved to %s", outputFile))
	return outputFile, err
}

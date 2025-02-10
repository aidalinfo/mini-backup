package backup

import (
	"fmt"
	"mini-backup/pkg/utils"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// BackupMySQL performs a MySQL dump using the mysqldump command-line tool.
func BackupMySQL(name string, config utils.Backup) ([]string, error) {
	if config.Mysql.Host == "" || config.Mysql.User == "" {
		return []string{}, fmt.Errorf("invalid MySQL configuration: missing required fields (Host: %s, User: %s)", config.Mysql.Host, config.Mysql.User)
	}

	// Format de la date pour nommer le dossier de backup
	date := time.Now().Format("20060102_150405")
	parentDir := fmt.Sprintf("%s/%s_mysql_backup_%s", config.Path.Local, name, date)

	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return []string{}, fmt.Errorf("failed to create backup parent directory: %w", err)
	}

	dumping := []string{}

	// Vérifie si allDatabases est activé
	if config.Mysql.AllDatabases {
		outputFile := filepath.Join(parentDir, fmt.Sprintf("%s-all_databases.sql", name))

		result, err := dumpAllDatabases(name, config, outputFile)
		if err != nil {
			return nil, fmt.Errorf("failed to dump all databases: %w", err)
		}

		dumping = append(dumping, result)
	} else {
		for _, database := range config.Mysql.Databases {
			outputFile := filepath.Join(parentDir, fmt.Sprintf("%s-%s.sql", name, database))

			result, err := dumpFunc(name, config, database, outputFile)
			if err != nil {
				fmt.Printf("Failed to dump database %s: %v\n", database, err)
				continue
			}
			dumping = append(dumping, result)
		}
	}

	return dumping, nil
}

// dumpAllDatabases executes mysqldump for all databases
func dumpAllDatabases(name string, config utils.Backup, outputFile string) (string, error) {
	cmd := exec.Command(
		"mysqldump",
		"-h", config.Mysql.Host,
		"-P", config.Mysql.Port,
		"-u", config.Mysql.User,
		"--ssl="+config.Mysql.SSL,
		"--all-databases",
		fmt.Sprintf("-p%s", config.Mysql.Password),
	)

	// Rediriger la sortie vers le fichier
	file, err := os.Create(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file for all databases: %w", err)
	}
	defer file.Close()

	cmd.Stdout = file

	// Exécuter la commande
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("mysqldump failed for all databases: %w", err)
	}

	fmt.Printf("Backup for all databases saved to %s\n", outputFile)
	return outputFile, nil
}

// dumpFunc executes mysqldump for a single database
func dumpFunc(name string, config utils.Backup, database, outputFile string) (string, error) {
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

	fmt.Printf("Backup saved to %s\n", outputFile)
	return outputFile, nil
}

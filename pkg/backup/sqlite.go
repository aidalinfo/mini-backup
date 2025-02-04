package backup

import (
	"fmt"
	"mini-backup/pkg/utils"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

func BackupSqlite(name string, config utils.Backup) (string, error) {
	logger.Info(fmt.Sprintf("Starting SQLite backup for: %s", name))

	// Ouvrir le fichier de base en lecture seule
	dbPath := config.Sqlite.Paths[0]
	dbFile, err := os.OpenFile(dbPath, os.O_RDONLY, 0666)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to open database file: %v", err))
		return "", err
	}
	defer dbFile.Close()

	// Acquisition d'un verrou exclusif sur le fichier de base
	if err = syscall.Flock(int(dbFile.Fd()), syscall.LOCK_EX); err != nil {
		logger.Error(fmt.Sprintf("Failed to lock database file: %v", err))
		return "", err
	}
	// Le verrou sera relâché automatiquement à la fermeture du fichier (ou via un defer explicite)
	defer syscall.Flock(int(dbFile.Fd()), syscall.LOCK_UN)

	// Construction du nom et chemin de sauvegarde
	backupName := fmt.Sprintf("%s_%s.bak", name, time.Now().Format("20060102_150405"))
	destinationPath := filepath.Join(config.Path.Local, backupName)

	// Construction de la commande sqlite3 pour réaliser la sauvegarde via la commande ".backup"
	backupCmd := fmt.Sprintf(".backup '%s'", destinationPath)
	cmd := exec.Command("sqlite3", dbPath, backupCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(fmt.Sprintf("Error executing sqlite3 backup command: %v, output: %s", err, string(output)))
		return "", err
	}

	logger.Info(fmt.Sprintf("Backup completed successfully: %s", destinationPath))
	return destinationPath, nil
}
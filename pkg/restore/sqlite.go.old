package restore

import (
	"fmt"
	"mini-backup/pkg/utils"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// RestoreSqlite restaure une base SQLite à partir d'un fichier de backup.
// La restauration s'effectue via la commande sqlite3 ".restore" et le fichier de base cible est verrouillé pendant l'opération.
func RestoreSqlite(name string, config utils.Backup, backupFilePath string) error {
	logger.Info(fmt.Sprintf("Starting SQLite restore for: %s", name))

	// On suppose que le chemin de la base cible est défini dans la configuration dans Sqlite.Paths[0]
	dbPath := config.Sqlite.Paths[0]

	// Vérification que le fichier de backup existe
	if _, err := os.Stat(backupFilePath); os.IsNotExist(err) {
		logger.Error(fmt.Sprintf("Backup file does not exist: %s", backupFilePath))
		return err
	}

	// Ouvrir le fichier de base cible en lecture-écriture
	dbFile, err := os.OpenFile(dbPath, os.O_RDWR, 0666)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to open database file: %v", err))
		return err
	}
	defer dbFile.Close()

	// Acquisition d'un verrou exclusif sur le fichier cible
	if err = syscall.Flock(int(dbFile.Fd()), syscall.LOCK_EX); err != nil {
		logger.Error(fmt.Sprintf("Failed to lock database file: %v", err))
		return err
	}
	defer syscall.Flock(int(dbFile.Fd()), syscall.LOCK_UN)

	// Construction de la commande sqlite3 pour restaurer le backup.
	// La commande exécutée sera : sqlite3 <dbPath> ".restore 'backupFilePath'"
	restoreCmd := fmt.Sprintf(".restore '%s'", backupFilePath)
	cmd := exec.Command("sqlite3", dbPath, restoreCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(fmt.Sprintf("Error executing sqlite3 restore command: %v, output: %s", err, string(output)))
		return err
	}

	logger.Info(fmt.Sprintf("Restore completed successfully for %s", filepath.Base(dbPath)))
	return nil
}
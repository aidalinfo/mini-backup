// file: handlers/logs.go
package handlers

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type BackupLogEntry struct {
	Timestamp  string `json:"timestamp"`
	BackupName string `json:"backupName"`
}

// LastBackupsFromLogs lit les logs et renvoie les 5 derniers backups "OK".
func LastBackupsFromLogs(c *fiber.Ctx) error {
	fmt.Println("GET /backups/last-logs")
	// 1. Déterminer le chemin du fichier de log depuis ta config ou en dur
	logFilePath := "logs/mini-backup.log"

	// 2. Lire tout le fichier
	file, err := os.Open(logFilePath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to open log file: %v", err),
		})
	}
	defer file.Close()

	// 3. Scanner ligne par ligne et extraire les backups OK
	entries := []BackupLogEntry{}
	scanner := bufio.NewScanner(file)

	// Exemple de pattern à chercher :
	// "INFO: 2023/05/23 10:46:01 main.go:15: [BOOTSTRAP_SERVER] [TRACING] : Backup OK : minio-data"
	// On veut la date/heure ET le nom du backup ("minio-data")
	//
	// Selon ton format, on peut utiliser une RegExp
	re := regexp.MustCompile(`(?P<date>\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}).*Backup OK\s*:\s*(?P<backup>\S+)`)

	for scanner.Scan() {
		line := scanner.Text()

		// On filtre seulement les lignes qui contiennent "Backup OK"
		if strings.Contains(line, "Backup OK") {
			// On essaie de matcher la regexp
			match := re.FindStringSubmatch(line)
			if len(match) > 2 {
				datePart := match[1]   // e.g. "2023/05/23 10:46:01"
				backupName := match[2] // e.g. "minio-data"

				// On peut garder la date brute, ou parser pour la formater
				// parse la date en time.Time
				parsedTime, parseErr := time.Parse("2006/01/02 15:04:05", datePart)
				// Si on veut un format custom
				var timeString string
				if parseErr == nil {
					timeString = parsedTime.Format(time.RFC3339)
				} else {
					timeString = datePart // fallback
				}

				entries = append(entries, BackupLogEntry{
					Timestamp:  timeString,
					BackupName: backupName,
				})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Error reading log file: %v", err),
		})
	}

	// 4. Récupérer les 5 dernières entrées
	const N = 5
	total := len(entries)
	if total > N {
		entries = entries[total-N:]
	}

	// 5. Retourner en JSON
	return c.JSON(fiber.Map{
		"backups": entries,
	})
}

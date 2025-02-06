package handlers

import (
	"fmt"
	"mini-backup/pkg/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/robfig/cron/v3"
)

// GetBackupConfig retourne la configuration d'un backup spécifique ou liste les backups du stockage distant.
func GetBackupConfig(c *fiber.Ctx) error {
	// Récupérer les paramètres de requête
	backupName := c.Query("name")
	if backupName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Parameter 'name' is required",
		})
	}

	remoteStorage := c.Query("remote_storage") // Paramètre optionnel

	// Charger la configuration
	config, err := utils.GetConfig()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load configuration",
		})
	}

	// Vérifier si le backup existe
	backupConfig, exists := config.Backups[backupName]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Backup not found",
		})
	}

	// Si le paramètre remote_storage est présent, lister les fichiers du stockage distant
	if remoteStorage != "" {
		s3client, err := utils.ManagerStorageFunc()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to initialize remote storage client",
			})
		}

		files, err := s3client.ListBackups(backupConfig.Path.S3)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to list backups from remote storage",
			})
		}

		return c.JSON(fiber.Map{
			"remote_backups": files,
		})
	}

	// Retourner la configuration locale si remote_storage n'est pas utilisé
	return c.JSON(fiber.Map{
		"name":   backupName,
		"config": backupConfig,
	})
}
// GetNextBackup retourne le prochain backup prévu
func GetNextBackup(c *fiber.Ctx) error {
	// Charger la configuration
	config, err := utils.GetConfig()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to load configuration: %v", err),
		})
	}

	// Trouver le prochain backup
	nextBackupName, nextBackupTime, err := findNextBackup(config)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to find next scheduled backup: %v", err),
		})
	}

	if nextBackupName == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No scheduled backup found",
		})
	}

	// Retourner la réponse
	return c.JSON(fiber.Map{
		"name": nextBackupName,
		"time": nextBackupTime.Format(time.RFC3339),
	})
}

// findNextBackup retourne le prochain backup à exécuter
func findNextBackup(config *utils.BackupConfig) (string, time.Time, error) {
	var nextBackupName string
	earliest := time.Time{}

	for name, backup := range config.Backups {
		if backup.Schedule.Standard != "" {
			nextTime, err := getNextCronExecution(backup.Schedule.Standard)
			if err != nil {
				return "", time.Time{}, err
			}

			// Met à jour le backup le plus proche
			if earliest.IsZero() || nextTime.Before(earliest) {
				earliest = nextTime
				nextBackupName = name
			}
		}
	}

	if nextBackupName == "" {
		return "", time.Time{}, nil
	}

	return nextBackupName, earliest, nil
}

// getNextCronExecution calcule la prochaine exécution d'une expression cron
func getNextCronExecution(cronExpression string) (time.Time, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cronExpression)
	if err != nil {
		return time.Time{}, err
	}
	return schedule.Next(time.Now()), nil
}
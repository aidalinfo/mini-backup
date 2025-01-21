package handlers

import (
	"mini-backup/pkg/utils"

	"github.com/gofiber/fiber/v2"
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

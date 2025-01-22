package handlers

import (
	"fmt"
	"mini-backup/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// ListBackups retourne la liste des backups disponibles.
func ListBackups(c *fiber.Ctx) error {
	config, err := utils.GetConfig()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load configuration",
		})
	}

	var backups []string
	for name := range config.Backups {
		backups = append(backups, name)
	}

	return c.JSON(fiber.Map{
		"backups": backups,
	})
}

func DetailBackup(c *fiber.Ctx) error {
	config, err := utils.GetConfig()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load configuration",
		})
	}
	return c.JSON(fiber.Map{
		"backup": config.Backups,
	})
}

// ListFilesForBackup retourne les fichiers pour un backup spécifique.
func ListFilesForBackup(c *fiber.Ctx) error {
	// Récupérer le nom du backup depuis les paramètres de route
	name := c.Params("name")

	// Charger la configuration
	config, err := utils.GetConfig()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load configuration",
		})
	}

	// Vérifier si le backup existe
	backupConfig, exists := config.Backups[name]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fmt.Sprintf("Backup '%s' not found", name),
		})
	}

	// Charger la configuration du serveur
	configServer, err := utils.GetConfigServer()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load server configuration",
		})
	}

	// Vérifier si RStorage contient au moins un élément
	if len(configServer.RStorage) == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "No storage configuration found",
		})
	}

	// Obtenir le premier élément de RStorage
	var firstStorageName string
	var firstStorageConfig utils.RStorageConfig
	for name, config := range configServer.RStorage {
		firstStorageName = name
		firstStorageConfig = config
		break
	}

	// Initialiser le client S3 avec le premier storage
	s3client, err := utils.RstorageManager(firstStorageName, &firstStorageConfig)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to initialize S3 manager: %v", err),
		})
	}

	// Lister les fichiers pour le chemin S3 du backup
	files, err := s3client.ListBackups(backupConfig.Path.S3)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to list backups for '%s': %v", name, err),
		})
	}

	// Retourner la liste des fichiers
	return c.JSON(fiber.Map{
		"backup": name,
		"files":  files,
	})
}
func ListFilesForAllBackup(c *fiber.Ctx) error {
	logger := utils.LoggerFunc()

	// Charger la configuration
	config, err := utils.GetConfig()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load configuration",
		})
	}
	configServer, err := utils.GetConfigServer()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get config server: %v", err), "[API] [HANDLER SERVER]")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load configuration",
		})
	}

	// Pour chaque stockage
	allFiles := make(map[string][]utils.BackupDetails)
	for name, configServer := range configServer.RStorage {
		s3client, err := utils.RstorageManager(name, &configServer)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to get storage manager: %v", err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to get storage manager: %v", err),
			})
		}

		// Pour chaque backup
		for backupName, backupConfig := range config.Backups {
			files, err := s3client.ListBackupsApi(backupConfig.Path.S3)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to list backups for '%s': %v", backupName, err), "[API] [HANDLER SERVER]")
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Failed to list backups for '%s': %v", backupName, err),
				})
			}
			// Ajouter les fichiers à la réponse
			allFiles[backupName] = files
		}
	}

	// Si aucun fichier trouvé
	if len(allFiles) == 0 {
		return c.JSON(fiber.Map{
			"files": "No files found",
		})
	}

	// Réponse avec les fichiers et leurs détails
	return c.JSON(fiber.Map{
		"files": allFiles,
	})
}

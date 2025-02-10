package handlers

import (
	"fmt"
	"mini-backup/pkg/utils"
	"net/url"

	"github.com/gofiber/fiber/v2"
)

func DownloadBackup(c *fiber.Ctx) error {
	logger := utils.LoggerFunc()
	logger.Info(fmt.Sprintf("DownloadBackup file : %s", c.Params("file")))
	fileName, err := url.QueryUnescape(c.Params("file"))
	if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nom du fichier invalide"})
	}
	configServer, err := utils.GetConfigServer()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get config server: %v", err), "[API] [HANDLER SERVER]")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load configuration",
		})
	}

	// Obtenir la première configuration de stockage disponible
	var firstStorageName string
	var firstStorageConfig utils.RStorageConfig
	for name, config := range configServer.RStorage {
		firstStorageName = name
		firstStorageConfig = config
		break
	}

	// Initialiser le client S3
	s3client, err := utils.RstorageManager(firstStorageName, &firstStorageConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to initialize S3 manager : %v", err), "[API] [HANDLER PACKAGE]")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to initialize S3 manager: %v", err),
		})
	}

	logger.Info(fmt.Sprintf("DownloadBackup file : %s", fileName))
	// Télécharger et déchiffrer le fichier
	decryptedData, err := s3client.DownloadAndDecrypt(fileName)
	if err != nil {
		logger.Error(fmt.Sprintf("Impossible de télécharger/déchiffrer le fichier : %v", err), "[API] [HANDLER PACKAGE]")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Impossible de télécharger/déchiffrer le fichier: %v", err)})
	}

	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))

	c.Set("Content-Type", "application/octet-stream")

	// Envoyer les données déchiffrées au client
	return c.Send(decryptedData)
}
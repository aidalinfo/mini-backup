package handlers

import (
	"fmt"
	"mini-backup/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

func GetConfigServer(c *fiber.Ctx) error {
	config, err := utils.GetConfigServer()
	logger := utils.LoggerFunc()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get config server: %v", err), "[API] [HANDLER SERVER]")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load configuration",
		})
	}
	logger.Debug(fmt.Sprintf("Loaded configuration: %v", config), "[API] [HANDLER SERVER]")
	return c.JSON(fiber.Map{
		"server": config,
	})
}

// GetRStorageCount retourne le nombre de configurations RStorage
func GetRStorageCount(c *fiber.Ctx) error {
	// Charger la configuration serveur
	config, err := utils.GetConfigServer()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load server configuration: " + err.Error(),
		})
	}

	// Compter les entrées RStorage
	count := len(config.RStorage)

	// Retourner la réponse
	return c.JSON(fiber.Map{
		"rstorage_count": count,
	})
}
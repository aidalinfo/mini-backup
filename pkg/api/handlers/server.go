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

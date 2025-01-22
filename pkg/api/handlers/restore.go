package handlers

import (
	"fmt"
	"log"
	"mini-backup/pkg/restore"
	"mini-backup/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

func RestoreBackup(c *fiber.Ctx) error {
	// Récupérer le paramètre `name` depuis la route
	name := c.Params("name")
	logger := utils.LoggerFunc()
	logger.Info(fmt.Sprintf("RestoreBackup : %s", name), "SOURCE API")
	// Appeler la fonction de restauration avec le nom
	err := restore.CoreRestore(name, "last")
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur de restauration : %v", err), "SOURCE API")
		log.Printf("Erreur de restauration : %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to restore backup", err,
		})
	}

	return c.JSON(fiber.Map{
		"message": "Backup restored successfully",
		"name":    name,
	})
}

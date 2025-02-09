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
	// Définir une structure pour récupérer le payload JSON
	type RestoreRequest struct {
		PathFile string `json:"pathFile"`
	}

	var req RestoreRequest
	// Parser le corps de la requête dans la structure
	if err := c.BodyParser(&req); err != nil {
		logger.Error(fmt.Sprintf("Erreur lors du parsing du payload : %v", err), "SOURCE API")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Payload invalide",
		})
	}
	logger.Info(fmt.Sprintf("RestoreBackup file : %s", req.PathFile), "SOURCE API")
	logger.Info(fmt.Sprintf("RestoreBackup : %s", name), "SOURCE API")
	// Appeler la fonction de restauration avec le nom
	err := restore.CoreRestore(name, req.PathFile, "", "")
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur de restauration : %v", err), "SOURCE API")
		log.Printf("Erreur de restauration : %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to restore backup: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Backup restored successfully",
		"name":    name,
	})
}

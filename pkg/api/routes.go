package api

import (
	"mini-backup/pkg/api/handlers"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configure les routes de l'application.
func SetupRoutes(app *fiber.App) {
	api := app.Group("/api")
	// Route pour récupérer la configuration d'un backup
	api.Get("/backup", handlers.GetBackupConfig)
	// Route pour lister les backups
	api.Get("/backups", handlers.ListBackups)
	// Route pour obtenir les backups détaillés
	api.Get("/backups/all", handlers.DetailBackup)
	api.Post("/restore/:name", handlers.RestoreBackup)
	// Route pour lister les fichiers d'un backup
	api.Get("/backups/:name/files", handlers.ListFilesForBackup)
	// api.Get("/backups/:name/list", handlers.ListBackupDetails)

	api.Get("/server/config", handlers.GetConfigServer)

	api.Get("/server/backups/list", handlers.ListFilesForAllBackup)
	api.Get("/backups/last-logs", handlers.LastBackupsFromLogs)

}

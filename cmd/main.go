package main

import (
	"fmt"
	"mini-backup/pkg/api"
	"mini-backup/pkg/backup"
	"mini-backup/pkg/utils"
)

func main() {

	logger := utils.LoggerFunc()

	if utils.GetEnv[bool]("AUTO_CONFIG") || utils.GetEnv[string]("AUTO_CONFIG") == "true" {
		err := utils.AutoConfigurationFunc()
		if err != nil {
			logger.Error(fmt.Sprintf("Erreur lors de la configuration automatique : %v", err))
			return
		}
	}
	modules, err := utils.LoadModules()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load modules: %v", err))
	}
	logger.Info(fmt.Sprintf("Loaded modules: %v", modules), utils.Bootstrap_server)
	serverConfig, err := utils.GetConfigServer()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load configuration: %v", err))
		return
	}
	logger.Info("Starting backup tool", serverConfig.Server.Port)
	defer logger.Close()

	logger.Info("Starting backup tool", utils.Bootstrap_server)
	app := api.ApiServer()

	go func() {
		// log.Println("API server running at http://localhost:" + serverConfig.Server.Port)
		logger.Info(fmt.Sprintf("API server running at http://localhost:%s", serverConfig.Server.Port), utils.Bootstrap_server)
		if err := app.Listen(":" + serverConfig.Server.Port); err != nil {
			logger.Error(fmt.Sprintf("Failed to start API server: %v", err))
			logger.Error(fmt.Sprintf("Failed to start API server: %v", err))
		}
	}()
	// Load configuration
	config, err := utils.GetConfig()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load configuration: %v", err), utils.Bootstrap_server)
		return
	}
	logger.Debug(fmt.Sprintf("Loaded configuration: %v", config), utils.Bootstrap_server)
	// Initialize scheduler
	scheduler := utils.NewScheduler()
	defer scheduler.Stop()

	// Schedule backups
	for name, backupConfig := range config.Backups {
		if backupConfig.Schedule.Standard != "" {
			logger.Info(fmt.Sprintf("Scheduling standard backup for %s: %s", name, backupConfig.Schedule.Standard), utils.Bootstrap_server)
			err := scheduler.AddJob(backupConfig.Schedule.Standard, func() {
				logger.Info(fmt.Sprintf("Executing standard backup for %s", name), utils.Bootstrap_server)
				backup.CoreBackup(name, false)
			})
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to schedule standard backup for %s: %v", name, err), utils.Bootstrap_server)
			}
		} else {
			logger.Info(fmt.Sprintf("No schedule found for %s, executing backup immediately", name), utils.Bootstrap_server)
			backup.CoreBackup(name, false)
		}

		if backupConfig.Schedule.Glacier != "" {
			logger.Info(fmt.Sprintf("Scheduling Glacier backup for %s: %s", name, backupConfig.Schedule.Glacier), utils.Bootstrap_server)
			err := scheduler.AddJob(backupConfig.Schedule.Glacier, func() {
				logger.Info(fmt.Sprintf("Executing Glacier backup for %s", name), utils.Bootstrap_server)
				backup.CoreBackup(name, true)
			})
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to schedule Glacier backup for %s: %v", name, err), utils.Bootstrap_server)
			}
		}
	}

	// Start the scheduler
	scheduler.Start()

	// Keep the program running
	select {}
}

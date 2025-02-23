package bootstrap

import (
	"fmt"
	"mini-backup/pkg/utils"
	"mini-backup/pkg/utils/server/packager"
)

var logger = utils.LoggerFunc()

func BootstrapModule() {
	config, err := utils.GetConfigServer()
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la lecture de la configuration : %v", err))
		return
	}

	localModules, err := utils.LoadModules()
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors du chargement des modules locaux : %v", err))
		return
	}

	modulesList, err := packager.ListModules()
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la récupération de la liste des modules : %v", err))
		return
	}

	for _, moduleName := range config.Modules {
		var moduleFound bool
		// Vérification si le module est déjà installé 
		if localMod, ok := localModules[moduleName]; ok {
			remoteModule := utils.Module{
				Name:    moduleName,
				Version:  localMod.Version,
				Type:     localMod.Type,
			}
			needUpdate, err := packager.CheckModuleVersion(remoteModule)
			if err != nil {
				logger.Error(fmt.Sprintf("Erreur lors de la vérification de la version du module %s : %v", moduleName, err))
			} else if needUpdate {
				// Lancement de l'installation pour mettre à jour le module
				for _, mod := range modulesList {
					if mod.Name == moduleName {
						modulePkg := packager.ModulePackage{
							Category:    mod.Category,
							Name:        mod.Name,
							ModuleInfo:  mod.ModuleInfo,
							DownloadURL: mod.DownloadURL,
						}
						logger.Info(fmt.Sprintf("Mise à jour du module %s...", mod.Name))
						if err := packager.InstallModule(modulePkg); err != nil {
							logger.Error(fmt.Sprintf("Erreur lors de la mise à jour du module %s : %v", mod.Name, err))
						} else {
							logger.Info(fmt.Sprintf("Module %s mis à jour avec succès", mod.Name))
						}
						break
					}
				}
			}
			moduleFound = true
		}

		// Si le module n'est pas installé localement, le télécharger et l'installer
		if !moduleFound {
			for _, mod := range modulesList {
				if mod.Name == moduleName {
					modulePkg := packager.ModulePackage{
						Category:    mod.Category,
						Name:        mod.Name,
						ModuleInfo:  mod.ModuleInfo,
						DownloadURL: mod.DownloadURL,
					}
					logger.Info(fmt.Sprintf("Installation du module %s...", mod.Name))
					if err := packager.InstallModule(modulePkg); err != nil {
						logger.Error(fmt.Sprintf("Erreur lors de l'installation du module %s : %v", mod.Name, err))
					} else {
						logger.Info(fmt.Sprintf("Module %s installé avec succès", mod.Name))
					}
					moduleFound = true
					break
				}
			}
			if !moduleFound {
				logger.Error(fmt.Sprintf("Module %s introuvable", moduleName))
			}
		}

	}
}

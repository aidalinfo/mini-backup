package bootstrap

import (
	"fmt"
	"log"
	"mini-backup/pkg/utils"
	"mini-backup/pkg/utils/server/packager"
	"os"
	"path/filepath"
)

func BootstrapModule() {
	config, err := utils.GetConfigServer()
	if err != nil {
		log.Fatalf("Erreur lors du chargement de la configuration: %v", err)
	}

	// Chargement des modules locaux (fichiers module.yaml)
	localModules, err := utils.LoadModules()
	if err != nil {
		log.Fatalf("Erreur lors du chargement des modules locaux: %v", err)
	}

	// Récupération de la liste complète des modules distants depuis l'index
	modulesList, err := packager.ListModules()
	if err != nil {
		log.Fatalf("Erreur lors de la récupération de l'index des modules: %v", err)
	}

	// Pour chaque module défini dans la configuration
	for _, moduleName := range config.Modules {
		var moduleFound bool

		// D'abord, on vérifie si le module est déjà installé localement
		if localMod, ok := localModules[moduleName]; ok {
			// Conversion du module local (utils.Module) en module compatible avec packager.CheckModuleVersion.
			// Ici, on suppose que le champ Name de utils.Module correspond à la clé attendue.
			remoteModule := utils.Module{
				Version:  localMod.Version,
				Type:     localMod.Type,
			}

			// Vérification de la version locale par rapport à l'index distant
			if err := packager.CheckModuleVersion(remoteModule); err != nil {
				fmt.Printf("Erreur lors de la vérification du module %s : %v\n", localMod.Name, err)
			}
			moduleFound = true
		}

		// Si le module n'est pas installé localement, le télécharger
		if !moduleFound {
			for _, mod := range modulesList {
				if mod.Name == moduleName {
					// Chemin pour enregistrer le fichier zip téléchargé
					zipPath := filepath.Join("modules", mod.Category, mod.Name+".zip")
					fmt.Printf("Téléchargement du module %s depuis %s...\n", mod.Name, mod.DownloadURL)
					if err := packager.DownloadModule(mod.DownloadURL, zipPath); err != nil {
						fmt.Printf("Erreur lors du téléchargement du module %s: %v\n", mod.Name, err)
					} else {
						fmt.Printf("Module %s téléchargé avec succès dans %s\n", mod.Name, zipPath)
						// Définir le dossier de destination pour la décompression
						unzipDest := filepath.Join("modules", mod.Category, mod.Name)
						fmt.Printf("Décompression du module %s dans %s...\n", mod.Name, unzipDest)
						if err := packager.UnzipModule(zipPath, unzipDest); err != nil {
							fmt.Printf("Erreur lors de la décompression du module %s: %v\n", mod.Name, err)
						} else {
							fmt.Printf("Module %s décompressé avec succès dans %s\n", mod.Name, unzipDest)
						}
						// Suppression du zip téléchargé
						if err := os.Remove(zipPath); err != nil {
							fmt.Printf("Erreur lors de la suppression du fichier zip %s: %v\n", zipPath, err)
						}
					}
					moduleFound = true
					break // On passe au module suivant dès qu'on a traité le module trouvé
				}
			}
			if !moduleFound {
				fmt.Printf("Module '%s' introuvable dans l'index des modules.\n", moduleName)
			}
		}
	}
}

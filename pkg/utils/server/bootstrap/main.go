package bootstrap

import (
	"fmt"
	"log"
	"mini-backup/pkg/utils"
	"mini-backup/pkg/utils/server/packager"
	"path/filepath"
)

func BootstrapModule() {
	config, err := utils.GetConfigServer()
	if err != nil {
		log.Fatalf("Erreur lors du chargement de la configuration: %v", err)
	}

	// Si des modules sont spécifiés dans la configuration, les traiter
	if len(config.Modules) > 0 {
		fmt.Println("Téléchargement des modules spécifiés dans la configuration:")

		// Récupérer la liste complète des modules depuis l'index du dépôt
		modulesList, err := packager.ListModules()
		if err != nil {
			log.Fatalf("Erreur lors de la récupération de l'index des modules: %v", err)
		}

		// Pour chaque module défini dans la configuration
		for _, moduleName := range config.Modules {
			var moduleFound bool
			for _, mod := range modulesList {
				if mod.Name == moduleName {
					moduleFound = true
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
					}
					break // On passe au module suivant dès qu'on a traité le module trouvé
				}
			}
			if !moduleFound {
				fmt.Printf("Module '%s' introuvable dans l'index des modules.\n", moduleName)
			}
		}
	} else {
		fmt.Println("Aucun module à télécharger dans la configuration.")
	}

	// Ici, poursuivre le bootstrap de votre serveur...
}

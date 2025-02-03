package commands

import (
	"fmt"
	"mini-backup/pkg/restore"
	"mini-backup/pkg/utils"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// NewRestoreCommand crée la commande CLI pour la restauration
func NewRestoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore [name] [version]",
		Short: "Restore a backup",
		Args:  cobra.MinimumNArgs(1), // Minimum 1 argument requis : le nom
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			version := "last" // Par défaut, la dernière version

			// Si une version est précisée, on la récupère
			if len(args) > 1 {
				version = args[1]
			}

			// Charger la configuration principale pour les backups
			config, err := utils.GetConfig()
			if err != nil {
				fmt.Printf("Erreur lors du chargement de la configuration principale : %v\n", err)
				return
			}

			// Vérifier que la configuration de backup existe pour ce nom
			backupConfig, ok := config.Backups[name]
			if !ok {
				fmt.Printf("Aucune configuration de backup trouvée pour : %s\n", name)
				return
			}

			// Charger la configuration de restauration
			restoresConfig, err := utils.GetRestoreConfig()
			if err != nil {
				fmt.Printf("Erreur lors du chargement des configurations de restauration : %v\n", err)
				return
			}

			// Filtrer les configurations de restauration par type
			fmt.Printf("Filtrage des configurations de restauration pour le type : %s\n", backupConfig.Type)
			filteredConfigs := make([]string, 0)
			for key, value := range restoresConfig.Restores {
				if restoreConfig, ok := value.(map[string]interface{}); ok {
					if restoreType, ok := restoreConfig["type"].(string); ok && restoreType == backupConfig.Type {
						filteredConfigs = append(filteredConfigs, key)
					}
				}
			}

			// Vérifier s'il y a des configurations correspondantes
			if len(filteredConfigs) == 0 {
				fmt.Printf("Aucune configuration de restauration trouvée pour le type : %s\n", backupConfig.Type)
				return
			}

			// Afficher les configurations disponibles
			fmt.Println("Configurations de restauration disponibles :")
			for i, configName := range filteredConfigs {
				fmt.Printf("%d. %s\n", i+1, configName)
			}

			// Demander à l'utilisateur de choisir une configuration
			fmt.Print("Sélectionnez une configuration de restauration (par défaut: 1) : ")
			var configChoice int
			fmt.Scanln(&configChoice)

			if configChoice < 1 || configChoice > len(filteredConfigs) {
				configChoice = 1 // Par défaut, prendre la première configuration
			}

			selectedConfig := filteredConfigs[configChoice-1]

			// Si aucune version n'est précisée, demander à l'utilisateur
			if len(args) == 1 && version == "last" {
				fmt.Printf("Voulez-vous restaurer le dernier backup pour %s ? (y/n) : ", name)
				var response string
				fmt.Scanln(&response)

				if strings.ToLower(response) == "n" {
					// Charger la configuration du serveur
					configServer, err := utils.GetConfigServer()
					if err != nil {
						fmt.Printf("Erreur lors du chargement de la configuration du serveur : %v\n", err)
						return
					}

					// Obtenir le premier storage disponible
					var firstStorageName string
					var firstStorageConfig utils.RStorageConfig
					for name, config := range configServer.RStorage {
						firstStorageName = name
						firstStorageConfig = config
						break
					}

					// Initialiser le client S3 avec la nouvelle méthode
					s3client, err := utils.RstorageManager(firstStorageName, &firstStorageConfig)
					if err != nil {
						fmt.Printf("Erreur lors de l'initialisation du gestionnaire S3 : %v\n", err)
						return
					}

					// Lister les backups disponibles
					fmt.Println("Liste des backups disponibles :")
					files, err := s3client.ListBackups(backupConfig.Path.S3)
					if err != nil {
						fmt.Printf("Erreur lors de la liste des backups : %v\n", err)
						return
					}

					// Afficher les fichiers disponibles
					for i, file := range files {
						fmt.Printf("%d. %s\n", i+1, file)
					}

					// Demander à l'utilisateur de choisir un fichier
					fmt.Print("Sélectionnez un numéro de backup : ")
					var choice int
					fmt.Scanln(&choice)

					// Valider le choix de l'utilisateur
					if choice < 1 || choice > len(files) {
						fmt.Println("Choix invalide, opération annulée.")
						return
					}

					version = filepath.Base(files[choice-1])
				}
			}

			// Appeler CoreRestore avec la configuration sélectionnée et la version choisie
			err = restore.CoreRestore(name, version, selectedConfig, "")
			if err != nil {
				fmt.Printf("Erreur lors de la restauration : %v\n", err)
			} else {
				fmt.Printf("Restauration réussie pour %s (version: %s, configuration: %s)\n", name, version, selectedConfig)
			}
		},
	}

	return cmd
}

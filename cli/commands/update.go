package commands

import (
	"fmt"
	"mini-backup/pkg/utils"

	"github.com/spf13/cobra"
)

var currentVersion = "1.0.0"

// NewUpdateCommand crée la commande CLI pour les mises à jour
func NewUpdateCommand(currentVersion string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Met à jour le logiciel (CLI et/ou serveur)",
		Run: func(cmd *cobra.Command, args []string) {
			latestVersion, err := utils.CheckForUpdates(currentVersion)
			if err != nil {
				fmt.Printf("Erreur lors de la vérification des mises à jour : %v\n", err)
				return
			}

			if latestVersion == "" {
				fmt.Println("Aucune mise à jour disponible.")
				return
			}

			server, _ := cmd.Flags().GetBool("server")
			cli, _ := cmd.Flags().GetBool("cli")

			// Met à jour uniquement le serveur si le flag est défini
			if server {
				if err := utils.UpdateServer(latestVersion); err != nil {
					fmt.Printf("Erreur lors de la mise à jour du serveur : %v\n", err)
				}
			}

			// Met à jour uniquement la CLI si le flag est défini
			if cli {
				if err := utils.UpdateCLI(latestVersion); err != nil {
					fmt.Printf("Erreur lors de la mise à jour de la CLI : %v\n", err)
				}
			}

			// Met à jour les deux si aucun flag spécifique n'est défini
			if !server && !cli {
				if err := utils.UpdateServer(latestVersion); err != nil {
					fmt.Printf("Erreur lors de la mise à jour du serveur : %v\n", err)
				}
				if err := utils.UpdateCLI(latestVersion); err != nil {
					fmt.Printf("Erreur lors de la mise à jour de la CLI : %v\n", err)
				}
			}
		},
	}

	// Ajout des flags pour spécifier les composants à mettre à jour
	cmd.Flags().Bool("server", false, "Met à jour le serveur uniquement")
	cmd.Flags().Bool("cli", false, "Met à jour la CLI uniquement")

	return cmd
}

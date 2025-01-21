package main

import (
	"fmt"
	"mini-backup/cli/commands"
	"os"

	"github.com/spf13/cobra"
)

var currentVersion = "0.1.0"

func main() {
	// Déclaration de la commande root
	var rootCmd = &cobra.Command{
		Use:   "cli",
		Short: "Mini Backup cli",
		Long:  `A CLI tool for managing backups and restores.`,
	}

	// Ajouter les commandes depuis les sous-packages
	rootCmd.AddCommand(commands.NewListCommand())
	rootCmd.AddCommand(commands.NewRestoreCommand())
	rootCmd.AddCommand(commands.NewUpdateCommand(currentVersion))

	// Exécuter la CLI
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

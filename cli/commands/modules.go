package commands

import (
	"fmt"
	"mini-backup/pkg/utils/server/packager"

	"github.com/spf13/cobra"
)

func NewModulesCommand() *cobra.Command {
	modulesCmd := &cobra.Command{
		Use:   "modules",
		Short: "Gestion des modules",
		Long:  "Liste et installe les modules disponibles",
	}

	modulesCmd.AddCommand(NewModulesListCommand())
	modulesCmd.AddCommand(NewModulesInstallCommand())

	return modulesCmd
}

func NewModulesListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Liste les modules disponibles",
		Run: func(cmd *cobra.Command, args []string) {
			modulesList, err := packager.ListModules()
			if err != nil {
				fmt.Printf("Erreur lors de la récupération de la liste des modules : %v\n", err)
				return
			}
			fmt.Println("Liste des modules disponibles :")
			for _, mod := range modulesList {
				// Affichage du nom, de la catégorie et de la version
				fmt.Printf("- %s (Catégorie : %s, Version : %s)\n", mod.Name, mod.Category, mod.ModuleInfo.Version)
			}
		},
	}
}

func NewModulesInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install [moduleName]",
		Short: "Installe un module",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			moduleName := args[0]
			modulesList, err := packager.ListModules()
			if err != nil {
				fmt.Printf("Erreur lors de la récupération de la liste des modules : %v\n", err)
				return
			}
			var found bool
			for _, mod := range modulesList {
				if mod.Name == moduleName {
					modulePkg := packager.ModulePackage{
						Category:    mod.Category,
						Name:        mod.Name,
						ModuleInfo:  mod.ModuleInfo,
						DownloadURL: mod.DownloadURL,
					}
					fmt.Printf("Installation du module %s...\n", moduleName)
					if err := packager.InstallModule(modulePkg); err != nil {
						fmt.Printf("Erreur lors de l'installation du module %s : %v\n", moduleName, err)
					} else {
						fmt.Printf("Module %s installé avec succès\n", moduleName)
					}
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("Module %s introuvable\n", moduleName)
			}
		},
	}
}

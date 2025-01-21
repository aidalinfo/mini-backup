package commands

import (
	"fmt"
	"mini-backup/pkg/utils"

	"github.com/spf13/cobra"
)

// NewListCommand retourne une nouvelle commande "list backup".
func NewListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list backup",
		Short: "List all available backups",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := utils.GetConfig()
			if err != nil {
				fmt.Printf("Failed to load configuration: %v\n", err)
				return
			}

			// fmt.Println("Available backups:")
			for name := range config.Backups {
				fmt.Printf("%s\n", name)
			}
		},
	}
}

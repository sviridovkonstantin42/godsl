package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Обновляет godsl",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("Обновил на версию: %s\n", "0.0.2")
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

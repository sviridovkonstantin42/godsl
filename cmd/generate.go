package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Транспилирует проект godsl->golang",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("Транспиляция пошла")
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

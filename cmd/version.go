package cmd

import (
	"fmt"

	"github.com/sviridovkonstantin42/godsl/internal/revision"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Версия godsl",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("Версия: %s\n", revision.Revision)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

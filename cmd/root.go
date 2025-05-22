package cmd

import (
	"fmt"
	"os"

	"github.com/sviridovkonstantin42/godsl/internal/revision"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "godsl",
	Short: "godsl is a simple CLI application",
	Long:  `godsl помогает улучшить опыт использования языка программирования Golang`,
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		noticeOfNewVersion()
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Используйте --help чтобы увидеть все доступные команды.")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func noticeOfNewVersion() {
	//Не проверяем версию для локальной сборки.
	if revision.Revision == "development" {
		return
	}
}

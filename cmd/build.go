package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build [path]",
	Short: "Транспилирует проект и выполняет go build в папке build",
	Run: func(_ *cobra.Command, args []string) {
		var projectPath string
		if len(args) > 0 {
			projectPath = args[0]
		}

		buildDir, err := generateProject(projectPath, "")
		if err != nil {
			fmt.Printf("Ошибка генерации: %v\n", err)
			return
		}

		cmd := exec.Command("go", "build", ".")
		cmd.Dir = buildDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			fmt.Printf("Ошибка go build: %v\n", err)
			return
		}

		fmt.Println("Сборка завершена успешно!")
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
}



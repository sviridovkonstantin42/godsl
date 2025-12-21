package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [path]",
	Short: "Транспилирует проект и выполняет go run в папке build",
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

		cmd := exec.Command("go", "run", ".")
		cmd.Dir = buildDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			fmt.Printf("Ошибка go run: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}



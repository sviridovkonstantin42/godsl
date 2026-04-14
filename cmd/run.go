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
	Run: func(cmd *cobra.Command, args []string) {
		var projectPath string
		if len(args) > 0 {
			projectPath = args[0]
		}

		clean, _ := cmd.Flags().GetBool("clean")
		watch, _ := cmd.Flags().GetBool("watch")

		if watch {
			watchRun(projectPath, clean)
			return
		}

		buildDir, err := generateProject(projectPath, "", GenerateOptions{Clean: clean})
		if err != nil {
			fmt.Printf("Ошибка генерации: %v\n", err)
			return
		}

		execDir := goBuildExecDir(buildDir, projectPath)

		execCmd := exec.Command("go", "run", ".")
		execCmd.Dir = execDir
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		if err := execCmd.Run(); err != nil {
			fmt.Printf("Ошибка go run: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().Bool("clean", false, "Полная пересборка build (без инкремента)")
	runCmd.Flags().BoolP("watch", "w", false, "Перезапускать при изменении .godsl файлов")
}

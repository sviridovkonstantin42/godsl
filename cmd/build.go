package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build [path]",
	Short: "Транспилирует проект и выполняет go build в папке build",
	Run: func(cmd *cobra.Command, args []string) {
		var projectPath string
		if len(args) > 0 {
			projectPath = args[0]
		}

		clean, _ := cmd.Flags().GetBool("clean")
		buildDir, err := generateProject(projectPath, "", GenerateOptions{Clean: clean})
		if err != nil {
			fmt.Printf("Ошибка генерации: %v\n", err)
			return
		}

		execDir := goBuildExecDir(buildDir, projectPath)

		execCmd := exec.Command("go", "build", ".")
		execCmd.Dir = execDir
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		if err := execCmd.Run(); err != nil {
			fmt.Printf("Ошибка go build: %v\n", err)
			return
		}

		fmt.Println("Сборка завершена успешно!")
	},
}

// goBuildExecDir возвращает директорию внутри buildDir, соответствующую projectPath.
// Например: buildDir=build, projectPath=./examples → build/examples
func goBuildExecDir(buildDir, projectPath string) string {
	if strings.TrimSpace(projectPath) == "" {
		return buildDir
	}
	cwd, err := os.Getwd()
	if err != nil {
		return buildDir
	}
	absProject, err := filepath.Abs(projectPath)
	if err != nil {
		return buildDir
	}
	rel, err := filepath.Rel(cwd, absProject)
	if err != nil || strings.HasPrefix(rel, "..") {
		return buildDir
	}
	return filepath.Join(buildDir, rel)
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().Bool("clean", false, "Полная пересборка build (без инкремента)")
}

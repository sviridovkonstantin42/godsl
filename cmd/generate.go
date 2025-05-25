package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sviridovkonstantin42/godsl/internal/transpiler"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Транспилирует проект godsl->golang",
	Run: func(_ *cobra.Command, args []string) {
		var projectPath *string
		if len(args) > 0 {
			projectPath = &args[0]
		}

		generate(projectPath)
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

func generate(path *string) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("Ошибка получения текущей директории:", err)
		return
	}

	if path != nil {
		dir = *path
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println("Ошибка чтения директории:", err)
		return
	}

	var godslFile string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".godsl" {
			godslFile = filepath.Join(dir, file.Name())
			break
		}
	}

	if godslFile == "" {
		fmt.Println("Файл с расширением .godsl не найден в указанной директории.")
		return
	}

	content, err := os.ReadFile(godslFile)
	if err != nil {
		fmt.Println("Ошибка чтения файла:", err)
		return
	}

	source := string(content)

	fmt.Println(transpiler.TranspileFile(source))
}

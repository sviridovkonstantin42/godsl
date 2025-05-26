package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sviridovkonstantin42/godsl/internal/transpiler"
	"golang.org/x/sync/errgroup"
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

type FileTask struct {
	SourcePath string
	TargetPath string
}

func generate(path *string) {
	rootDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Ошибка получения текущей директории:", err)
		return
	}

	if path != nil {
		rootDir = *path
	}

	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		fmt.Printf("Директория %s не существует\n", rootDir)
		return
	}

	buildDirPath, err := os.Getwd()
	if err != nil {
		fmt.Println("Ошибка получения текущей директории:", err)
		return
	}
	buildDir := filepath.Join(buildDirPath, "build")

	tasks, err := collectGodslFiles(rootDir, buildDir)
	if err != nil {
		fmt.Printf("Ошибка сбора файлов: %v\n", err)
		return
	}

	if len(tasks) == 0 {
		fmt.Println("Файлы с расширением .godsl не найдены в проекте.")
		return
	}

	fmt.Printf("Найдено %d файлов .godsl для транспиляции\n", len(tasks))

	if err := transpileFilesParallel(tasks, buildDir); err != nil {
		fmt.Printf("Ошибка транспиляции: %v\n", err)
		if buildExists, _ := doesBuildExist(buildDir); buildExists {
			fmt.Println("Удаление папки build из-за ошибок...")
			os.RemoveAll(buildDir)
		}
		return
	}

	fmt.Println("Транспиляция завершена успешно!")
}

func collectGodslFiles(rootDir, buildDir string) ([]FileTask, error) {
	var tasks []FileTask

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && info.Name() == "build" {
			return filepath.SkipDir
		}

		if !info.IsDir() && filepath.Ext(path) == ".godsl" {
			relPath, err := filepath.Rel(rootDir, path)
			if err != nil {
				return fmt.Errorf("ошибка вычисления относительного пути для %s: %v", path, err)
			}

			targetPath := filepath.Join(buildDir, strings.TrimSuffix(relPath, ".godsl")+".go")

			tasks = append(tasks, FileTask{
				SourcePath: path,
				TargetPath: targetPath,
			})
		}

		return nil
	})

	return tasks, err
}

func transpileFilesParallel(tasks []FileTask, buildDir string) error {
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории build: %v", err)
	}

	g := &errgroup.Group{}

	g.SetLimit(8)

	for _, task := range tasks {
		task := task
		g.Go(func() error {
			err := transpileFile(task)
			if err == nil {
				fmt.Printf("✓ %s -> %s\n", task.SourcePath, task.TargetPath)
			}
			return err
		})
	}

	return g.Wait()
}

func doesBuildExist(buildDir string) (bool, error) {
	_, err := os.Stat(buildDir)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func transpileFile(task FileTask) error {
	content, err := os.ReadFile(task.SourcePath)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла %s: %v", task.SourcePath, err)
	}

	source := string(content)
	transpiledCode, err := transpiler.TranspileFile(source)
	if err != nil {
		return fmt.Errorf("ошибка транспиляции файла %s: %v", task.SourcePath, err)
	}

	targetDir := filepath.Dir(task.TargetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории %s: %v", targetDir, err)
	}

	if err := os.WriteFile(task.TargetPath, []byte(transpiledCode), 0644); err != nil {
		return fmt.Errorf("ошибка записи файла %s: %v", task.TargetPath, err)
	}

	return nil
}

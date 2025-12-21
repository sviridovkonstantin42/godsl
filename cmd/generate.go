package cmd

import (
	"fmt"
	"io"
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
		var projectPath string
		if len(args) > 0 {
			projectPath = args[0]
		}
		if _, err := generateProject(projectPath, ""); err != nil {
			fmt.Printf("Ошибка: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

type FileTask struct {
	SourcePath string
	TargetPath string
}

// generateProject транспилирует проект (или файл) в buildDir и копирует все остальные файлы как есть.
// Возвращает абсолютный путь к buildDir.
func generateProject(projectPath string, buildDirOverride string) (string, error) {
	rootDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("ошибка получения текущей директории: %w", err)
	}

	if strings.TrimSpace(projectPath) != "" {
		rootDir = projectPath
	}
	rootDir, err = filepath.Abs(rootDir)
	if err != nil {
		return "", fmt.Errorf("ошибка получения абсолютного пути проекта: %w", err)
	}

	fi, err := os.Stat(rootDir)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("путь %s не существует", rootDir)
	}
	if err != nil {
		return "", fmt.Errorf("ошибка доступа к пути %s: %w", rootDir, err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("ошибка получения текущей директории: %w", err)
	}
	buildDir := filepath.Join(cwd, "build")
	if strings.TrimSpace(buildDirOverride) != "" {
		buildDir = buildDirOverride
	}
	buildDir, err = filepath.Abs(buildDir)
	if err != nil {
		return "", fmt.Errorf("ошибка получения абсолютного пути build: %w", err)
	}

	// Пересоздаем build, чтобы не оставлять мусор от предыдущей генерации.
	_ = os.RemoveAll(buildDir)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return "", fmt.Errorf("ошибка создания директории build: %w", err)
	}

	walkRoot := rootDir
	relBase := rootDir
	if !fi.IsDir() {
		// Если передали конкретный файл, то относительные пути считаем от его директории,
		// чтобы выход получался build/<имя_файла>.go, а не build/.go
		relBase = filepath.Dir(rootDir)
	}

	tasks, copyTasks, err := collectProjectTasks(walkRoot, relBase, buildDir)
	if err != nil {
		return "", fmt.Errorf("ошибка сбора файлов: %w", err)
	}

	if len(tasks) == 0 && len(copyTasks) == 0 {
		return "", fmt.Errorf("в проекте нет файлов для обработки")
	}

	if len(tasks) == 0 {
		fmt.Println("Файлы с расширением .godsl не найдены в проекте (будут только скопированы остальные файлы).")
	} else {
		fmt.Printf("Найдено %d файлов .godsl для транспиляции\n", len(tasks))
	}

	if err := copyFilesParallel(copyTasks); err != nil {
		_ = os.RemoveAll(buildDir)
		return "", fmt.Errorf("ошибка копирования файлов: %w", err)
	}

	if err := transpileFilesParallel(tasks); err != nil {
		_ = os.RemoveAll(buildDir)
		return "", fmt.Errorf("ошибка транспиляции: %w", err)
	}

	fmt.Println("Транспиляция завершена успешно!")
	return buildDir, nil
}

func collectProjectTasks(walkRoot, relBase, buildDir string) ([]FileTask, []FileTask, error) {
	var tasks []FileTask
	var copyTasks []FileTask

	err := filepath.Walk(walkRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && info.Name() == "build" {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(relBase, path)
		if err != nil {
			return fmt.Errorf("ошибка вычисления относительного пути для %s: %v", path, err)
		}

		if filepath.Ext(path) == ".godsl" {
			targetPath := filepath.Join(buildDir, strings.TrimSuffix(relPath, ".godsl")+".go")
			tasks = append(tasks, FileTask{SourcePath: path, TargetPath: targetPath})
			return nil
		}

		// Любой другой файл копируем как есть (go.mod/go.sum/ресурсы/и т.д.)
		copyTasks = append(copyTasks, FileTask{
			SourcePath: path,
			TargetPath: filepath.Join(buildDir, relPath),
		})
		return nil
	})

	return tasks, copyTasks, err
}

func transpileFilesParallel(tasks []FileTask) error {
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

func copyFilesParallel(tasks []FileTask) error {
	g := &errgroup.Group{}
	g.SetLimit(8)

	for _, task := range tasks {
		task := task
		g.Go(func() error {
			if err := copyFile(task.SourcePath, task.TargetPath); err != nil {
				return err
			}
			return nil
		})
	}
	return g.Wait()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла %s: %w", src, err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("ошибка создания директории %s: %w", filepath.Dir(dst), err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("ошибка создания файла %s: %w", dst, err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("ошибка копирования %s -> %s: %w", src, dst, err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("ошибка закрытия файла %s: %w", dst, err)
	}

	return nil
}

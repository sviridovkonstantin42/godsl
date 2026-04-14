package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [имя-приложения]",
	Short: "Инициализирует новый godsl проект",
	Long: `Создаёт структуру нового godsl проекта:

  <имя>/
    go.mod          — модуль Go
    main.godsl      — точка входа с примером try/catch
    .gitignore      — исключает build/ из git

Примеры:
  godsl init myapp         создать проект myapp в ./myapp/
  godsl init myapp --module github.com/user/myapp
  godsl init .             инициализировать в текущей директории`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		modulePath, _ := cmd.Flags().GetString("module")

		// Определяем целевую директорию и имя модуля
		targetDir := "."
		appName := ""

		if len(args) > 0 && args[0] != "." {
			targetDir = args[0]
			appName = filepath.Base(targetDir) // только имя директории, не полный путь
		} else {
			// Используем имя текущей директории
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("ошибка получения текущей директории: %w", err)
			}
			appName = filepath.Base(cwd)
		}

		// Путь модуля по умолчанию = имя приложения
		if modulePath == "" {
			modulePath = appName
		}

		absTarget, err := filepath.Abs(targetDir)
		if err != nil {
			return fmt.Errorf("неверный путь: %w", err)
		}

		// Проверяем: не инициализируем ли уже существующий проект
		if targetDir != "." {
			if _, err := os.Stat(absTarget); err == nil {
				return fmt.Errorf("директория %q уже существует", targetDir)
			}
		} else {
			// В текущей директории проверяем, нет ли уже go.mod
			if _, err := os.Stat(filepath.Join(absTarget, "go.mod")); err == nil {
				return fmt.Errorf("go.mod уже существует в текущей директории")
			}
		}

		// Создаём директорию
		if err := os.MkdirAll(absTarget, 0755); err != nil {
			return fmt.Errorf("ошибка создания директории: %w", err)
		}

		// Определяем версию Go
		goVersion := detectGoVersion()

		// Генерируем файлы
		files := []struct {
			rel     string
			content string
		}{
			{
				rel:     "go.mod",
				content: buildGoMod(modulePath, goVersion),
			},
			{
				rel:     "main.godsl",
				content: buildMainGodsl(appName),
			},
			{
				rel:     ".gitignore",
				content: "build/\n",
			},
		}

		for _, f := range files {
			path := filepath.Join(absTarget, f.rel)
			if err := os.WriteFile(path, []byte(f.content), 0644); err != nil {
				return fmt.Errorf("ошибка записи %s: %w", f.rel, err)
			}
			fmt.Printf("  создан  %s\n", filepath.Join(targetDir, f.rel))
		}

		fmt.Println()
		fmt.Printf("✓ Проект %q инициализирован!\n", appName)
		fmt.Println()
		fmt.Println("Следующие шаги:")

		if targetDir != "." {
			fmt.Printf("  cd %s\n", targetDir)
		}
		fmt.Println("  godsl run    — транспилировать и запустить")
		fmt.Println("  godsl fmt    — форматировать .godsl файлы")
		fmt.Println("  godsl build  — собрать бинарник")

		return nil
	},
}

// detectGoVersion пытается определить версию Go через `go env GOVERSION`.
// При ошибке возвращает разумный дефолт.
func detectGoVersion() string {
	out, err := exec.Command("go", "env", "GOVERSION").Output()
	if err != nil {
		return "1.22"
	}
	ver := strings.TrimSpace(string(out))
	// "go1.22.3" → "1.22"
	ver = strings.TrimPrefix(ver, "go")
	parts := strings.Split(ver, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return "1.22"
}

func buildGoMod(modulePath, goVersion string) string {
	return fmt.Sprintf("module %s\n\ngo %s\n", modulePath, goVersion)
}

func buildMainGodsl(appName string) string {
	return fmt.Sprintf(`package main

func main() {
}
`)
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().String("module", "", "Путь модуля Go (по умолчанию: имя приложения)")
}

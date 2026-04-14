package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sviridovkonstantin42/godsl/internal/transpiler"
)

var fmtCmd = &cobra.Command{
	Use:   "fmt [путь...]",
	Short: "Форматирует .godsl файлы (как gofmt, но с поддержкой try/catch)",
	Long: `Форматирует .godsl файлы в каноничный стиль.

Примеры:
  godsl fmt ./...        форматировать все .godsl файлы рекурсивно
  godsl fmt .            форматировать .godsl файлы в текущей директории
  godsl fmt main.godsl   форматировать конкретный файл
  godsl fmt --check ./... проверить форматирование без записи`,
	RunE: func(cmd *cobra.Command, args []string) error {
		checkOnly, _ := cmd.Flags().GetBool("check")
		listOnly, _ := cmd.Flags().GetBool("list")

		// По умолчанию — текущая директория рекурсивно
		patterns := args
		if len(patterns) == 0 {
			patterns = []string{"./..."}
		}

		files, err := collectGodslFiles(patterns)
		if err != nil {
			return err
		}

		if len(files) == 0 {
			fmt.Println("Нет .godsl файлов для форматирования.")
			return nil
		}

		var changed []string
		var errs []string

		for _, path := range files {
			original, err := os.ReadFile(path)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", path, err))
				continue
			}

			formatted, err := transpiler.FormatFile(string(original))
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", path, err))
				continue
			}

			if string(original) == formatted {
				continue // уже отформатирован
			}

			changed = append(changed, path)

			if listOnly || checkOnly {
				fmt.Println(path)
				continue
			}

			if err := os.WriteFile(path, []byte(formatted), 0644); err != nil {
				errs = append(errs, fmt.Sprintf("%s: ошибка записи: %v", path, err))
				continue
			}
			fmt.Println(path)
		}

		if len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintln(os.Stderr, e)
			}
		}

		if checkOnly && len(changed) > 0 {
			return fmt.Errorf("найдено %d файлов с неправильным форматированием", len(changed))
		}

		return nil
	},
}

// collectGodslFiles собирает пути ко всем .godsl файлам по заданным паттернам.
// Поддерживает ./..., ./pkg/..., конкретные файлы и директории.
func collectGodslFiles(patterns []string) ([]string, error) {
	seen := make(map[string]bool)
	var result []string

	for _, pattern := range patterns {
		// Паттерн ./... или dir/...
		recursive := false
		root := pattern
		if strings.HasSuffix(pattern, "/...") {
			recursive = true
			root = strings.TrimSuffix(pattern, "/...")
		} else if pattern == "..." {
			recursive = true
			root = "."
		}

		// Нормализуем путь
		if root == "" {
			root = "."
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("неверный путь %q: %w", pattern, err)
		}

		info, err := os.Stat(absRoot)
		if err != nil {
			return nil, fmt.Errorf("путь %q не существует: %w", pattern, err)
		}

		if !info.IsDir() {
			// Конкретный файл
			if filepath.Ext(absRoot) == ".godsl" && !seen[absRoot] {
				seen[absRoot] = true
				result = append(result, absRoot)
			}
			continue
		}

		if recursive {
			err = filepath.Walk(absRoot, func(path string, fi os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if fi.IsDir() && fi.Name() == "build" {
					return filepath.SkipDir
				}
				if !fi.IsDir() && filepath.Ext(path) == ".godsl" && !seen[path] {
					seen[path] = true
					result = append(result, path)
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("ошибка обхода %q: %w", absRoot, err)
			}
		} else {
			// Только файлы в директории (не рекурсивно)
			entries, err := os.ReadDir(absRoot)
			if err != nil {
				return nil, fmt.Errorf("ошибка чтения директории %q: %w", absRoot, err)
			}
			for _, e := range entries {
				if !e.IsDir() && filepath.Ext(e.Name()) == ".godsl" {
					path := filepath.Join(absRoot, e.Name())
					if !seen[path] {
						seen[path] = true
						result = append(result, path)
					}
				}
			}
		}
	}

	return result, nil
}

func init() {
	rootCmd.AddCommand(fmtCmd)
	fmtCmd.Flags().BoolP("check", "c", false, "Проверить форматирование без записи (ненулевой код выхода, если есть изменения)")
	fmtCmd.Flags().BoolP("list", "l", false, "Вывести список файлов с неправильным форматированием без записи")
}

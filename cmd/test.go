package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test [флаги godsl] [-- флаги и пакеты go test]",
	Short: "Транспилирует проект и запускает go test",
	Long: `Сначала транспилирует .godsl → .go, затем запускает go test.
Все аргументы после -- передаются напрямую в go test.

Примеры:
  godsl test                    транспилировать и запустить все тесты
  godsl test -- ./...           то же самое явно
  godsl test -- -v -run TestFoo запустить конкретный тест с флагом -v
  godsl test --clean -- ./...   пересобрать и запустить тесты`,

	// DisableFlagParsing позволяет принимать произвольные флаги go test
	// без их объявления в cobra. Разделитель -- передаёт управление go test.
	DisableFlagParsing: true,

	Run: func(cmd *cobra.Command, args []string) {
		// Разбираем args вручную:
		//   --clean     наш флаг
		//   --help / -h показываем справку
		//   --          всё после этого идёт в go test
		//   остальное   идёт в go test
		clean := false
		var goTestArgs []string
		passthrough := false

		for i := 0; i < len(args); i++ {
			a := args[i]
			switch {
			case passthrough:
				goTestArgs = append(goTestArgs, a)
			case a == "--":
				passthrough = true
			case a == "--clean":
				clean = true
			case a == "--help" || a == "-h":
				_ = cmd.Help()
				return
			default:
				// Всё остальное передаём go test (флаги вида -v, -run и пакеты)
				goTestArgs = append(goTestArgs, a)
			}
		}

		// По умолчанию тестируем всё
		if len(goTestArgs) == 0 {
			goTestArgs = []string{"./..."}
		}

		// Транспиляция
		buildDir, err := generateProject("", "", GenerateOptions{Clean: clean})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка генерации: %v\n", err)
			os.Exit(1)
		}

		// Маппинг пакетных паттернов: переводим пути из исходного дерева в build
		mappedArgs := mapPackagePatterns(goTestArgs, buildDir)

		fmt.Printf("Запуск: go test %s\n", strings.Join(mappedArgs, " "))

		execCmd := exec.Command("go", append([]string{"test"}, mappedArgs...)...)
		execCmd.Dir = buildDir
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		if err := execCmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "Ошибка go test: %v\n", err)
			os.Exit(1)
		}
	},
}

// mapPackagePatterns конвертирует пакетные паттерны из исходного дерева в build-дерево.
// Флаги (начинающиеся с -) остаются как есть.
// ./... → ./... (go test в buildDir и так найдёт все пакеты)
// ./pkg/subpkg → ./pkg/subpkg (сохраняем относительный путь)
func mapPackagePatterns(args []string, buildDir string) []string {
	cwd, err := os.Getwd()
	if err != nil {
		return args
	}

	var result []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			// Флаг go test — передаём как есть
			result = append(result, a)
			continue
		}

		// Паттерн пакета: пробуем смаппировать в build
		mapped := mapSinglePattern(a, cwd, buildDir)
		result = append(result, mapped)
	}
	return result
}

// mapSinglePattern переводит один паттерн пакета в путь относительно buildDir.
func mapSinglePattern(pattern, cwd, buildDir string) string {
	// ./... или ... — всё дерево, оставляем как есть
	if pattern == "./..." || pattern == "..." {
		return "./..."
	}

	// Если абсолютный путь или начинается с ./ — вычисляем rel от cwd
	if strings.HasPrefix(pattern, "./") || strings.HasPrefix(pattern, "/") {
		abs, err := filepath.Abs(pattern)
		if err != nil {
			return pattern
		}
		rel, err := filepath.Rel(cwd, abs)
		if err != nil || strings.HasPrefix(rel, "..") {
			return pattern
		}
		// Убеждаемся что директория существует в buildDir
		candidate := filepath.Join(buildDir, rel)
		if _, err := os.Stat(candidate); err == nil {
			return "./" + filepath.ToSlash(rel)
		}
	}

	// Иначе — пакетный путь модуля, передаём как есть (go test разберётся)
	return pattern
}

func init() {
	rootCmd.AddCommand(testCmd)
}

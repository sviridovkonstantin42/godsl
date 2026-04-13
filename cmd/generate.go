package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	Run: func(cmd *cobra.Command, args []string) {
		var projectPath string
		if len(args) > 0 {
			projectPath = args[0]
		}

		clean, _ := cmd.Flags().GetBool("clean")
		if _, err := generateProject(projectPath, "", GenerateOptions{Clean: clean}); err != nil {
			fmt.Printf("Ошибка: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().Bool("clean", false, "Полная пересборка build (без инкремента)")
}

type FileTask struct {
	SourcePath string
	TargetPath string
}

type GenerateOptions struct {
	Clean bool
}

const cacheFileName = ".godslcache.json"

type cacheEntry struct {
	TargetRel string `json:"targetRel"`
	Size      int64  `json:"size"`
	ModTime   int64  `json:"modTime"`
	Hash      string `json:"hash,omitempty"` // только для .godsl (чтобы избегать лишней транспиляции)
}

type buildCache struct {
	Version int                   `json:"version"`
	Godsl   map[string]cacheEntry `json:"godsl"` // key: relPath (.godsl)
	Files   map[string]cacheEntry `json:"files"` // key: relPath (non-.godsl)
}

// generateProject транспилирует проект (или файл) в buildDir и копирует все остальные файлы как есть.
// Возвращает абсолютный путь к buildDir.
func generateProject(projectPath string, buildDirOverride string, opts GenerateOptions) (string, error) {
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

	if opts.Clean {
		_ = os.RemoveAll(buildDir)
	}
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return "", fmt.Errorf("ошибка создания директории build: %w", err)
	}

	walkRoot := rootDir
	// relBase всегда берём от cwd, чтобы структура build/ отражала структуру проекта
	// независимо от того, передан ли конкретный путь (./examples) или нет.
	// Например: go run . build ./examples → build/examples/main.go, а не build/main.go.
	relBase := cwd
	if !fi.IsDir() {
		// Если передали конкретный файл — относительные пути от директории файла,
		// но корень по-прежнему cwd, чтобы build/ был предсказуемым.
		relBase = cwd
	}

	cache, err := loadBuildCache(buildDir)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения кэша: %w", err)
	}
	if opts.Clean {
		cache = newBuildCache()
	}

	tasks, copyTasks, deletions, cachedGodsl, cachedFiles, nextCache, err := planProjectTasks(walkRoot, relBase, buildDir, cache)
	if err != nil {
		return "", fmt.Errorf("ошибка сбора файлов: %w", err)
	}

	if len(tasks) == 0 && len(copyTasks) == 0 && len(deletions) == 0 {
		// Если в проекте вообще нет файлов (и нет кэша) — это ошибка.
		if len(nextCache.Godsl) == 0 && len(nextCache.Files) == 0 {
			return "", fmt.Errorf("в проекте нет файлов для обработки")
		}
		// Иначе просто ничего не изменилось — это ок.
		if len(cachedGodsl) > 0 {
			fmt.Printf("Нет изменений. Закэшировано .godsl файлов: %d\n", len(cachedGodsl))
			for _, p := range cachedGodsl {
				fmt.Printf("↺ cached %s\n", p)
			}
		} else {
			fmt.Println("Нет изменений.")
		}
		if err := saveBuildCache(buildDir, nextCache); err != nil {
			return "", fmt.Errorf("ошибка записи кэша: %w", err)
		}
		return buildDir, nil
	}

	if len(cachedGodsl) > 0 {
		fmt.Printf("Закэшировано (пропущено) .godsl файлов: %d\n", len(cachedGodsl))
		for _, p := range cachedGodsl {
			fmt.Printf("↺ cached %s\n", p)
		}
	}

	if len(deletions) > 0 {
		fmt.Printf("Удалено из build: %d файлов\n", len(deletions))
		for _, p := range deletions {
			_ = os.Remove(p)
			_ = cleanupEmptyDirs(buildDir, filepath.Dir(p))
		}
	}

	if len(copyTasks) > 0 {
		fmt.Printf("Копирование: %d файлов\n", len(copyTasks))
		if err := copyFilesParallel(copyTasks); err != nil {
			return "", fmt.Errorf("ошибка копирования файлов: %w", err)
		}
	}

	if len(tasks) > 0 {
		fmt.Printf("Транспиляция: %d файлов (.godsl)\n", len(tasks))
		if err := transpileFilesParallel(tasks); err != nil {
			return "", fmt.Errorf("ошибка транспиляции: %w", err)
		}
	}

	if err := saveBuildCache(buildDir, nextCache); err != nil {
		return "", fmt.Errorf("ошибка записи кэша: %w", err)
	}

	_ = cachedFiles // оставляем на будущее (например, детальный вывод по обычным файлам)
	fmt.Println("Готово (инкрементально).")
	return buildDir, nil
}

func planProjectTasks(walkRoot, relBase, buildDir string, cache buildCache) ([]FileTask, []FileTask, []string, []string, []string, buildCache, error) {
	var tasks []FileTask
	var copyTasks []FileTask
	var deletions []string
	var cachedGodsl []string
	var cachedFiles []string

	seenGodsl := make(map[string]bool)
	seenFiles := make(map[string]bool)
	nextCache := cache
	if nextCache.Godsl == nil {
		nextCache.Godsl = map[string]cacheEntry{}
	}
	if nextCache.Files == nil {
		nextCache.Files = map[string]cacheEntry{}
	}

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
		relPath = filepath.Clean(relPath)

		if filepath.Ext(path) == ".godsl" {
			targetRel := strings.TrimSuffix(relPath, ".godsl") + ".go"
			targetPath := filepath.Join(buildDir, targetRel)

			prev, ok := nextCache.Godsl[relPath]
			size := info.Size()
			modTime := info.ModTime().UnixNano()
			seenGodsl[relPath] = true

			// Быстрый skip по (size, mtime). Если они совпали — файл не трогаем.
			if ok && prev.Size == size && prev.ModTime == modTime {
				cachedGodsl = append(cachedGodsl, relPath)
				return nil
			}

			// Иначе хешируем содержимое (транспиляция тяжёлая, поэтому хотим избегать ложных срабатываний).
			h, err := fileSHA256Hex(path)
			if err != nil {
				return err
			}
			if ok && prev.Hash == h {
				// Контент тот же — обновим метаданные, но не транспилируем.
				nextCache.Godsl[relPath] = cacheEntry{TargetRel: targetRel, Size: size, ModTime: modTime, Hash: h}
				cachedGodsl = append(cachedGodsl, relPath)
				return nil
			}

			nextCache.Godsl[relPath] = cacheEntry{TargetRel: targetRel, Size: size, ModTime: modTime, Hash: h}
			tasks = append(tasks, FileTask{SourcePath: path, TargetPath: targetPath})
			return nil
		}

		// Любой другой файл копируем как есть (go.mod/go.sum/ресурсы/и т.д.)
		targetRel := relPath
		targetPath := filepath.Join(buildDir, targetRel)

		prev, ok := nextCache.Files[relPath]
		size := info.Size()
		modTime := info.ModTime().UnixNano()
		seenFiles[relPath] = true

		// Для обычных файлов достаточно (size,mtime) — если они совпали, пропускаем копирование.
		if ok && prev.Size == size && prev.ModTime == modTime {
			cachedFiles = append(cachedFiles, relPath)
			return nil
		}

		nextCache.Files[relPath] = cacheEntry{TargetRel: targetRel, Size: size, ModTime: modTime}
		copyTasks = append(copyTasks, FileTask{SourcePath: path, TargetPath: targetPath})
		return nil
	})

	if err != nil {
		return nil, nil, nil, nil, nil, buildCache{}, err
	}

	// Удаляем результаты для удалённых исходников.
	for rel, e := range cache.Godsl {
		if !seenGodsl[rel] {
			delete(nextCache.Godsl, rel)
			deletions = append(deletions, filepath.Join(buildDir, e.TargetRel))
		}
	}
	for rel, e := range cache.Files {
		if !seenFiles[rel] {
			delete(nextCache.Files, rel)
			deletions = append(deletions, filepath.Join(buildDir, e.TargetRel))
		}
	}

	return tasks, copyTasks, deletions, cachedGodsl, cachedFiles, nextCache, nil
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
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("ошибка stat файла %s: %w", src, err)
	}

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

	// Сохраняем права (насколько возможно)
	_ = os.Chmod(dst, srcInfo.Mode())

	return nil
}

func newBuildCache() buildCache {
	return buildCache{
		Version: 1,
		Godsl:   map[string]cacheEntry{},
		Files:   map[string]cacheEntry{},
	}
}

func loadBuildCache(buildDir string) (buildCache, error) {
	path := filepath.Join(buildDir, cacheFileName)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newBuildCache(), nil
		}
		return buildCache{}, err
	}

	var c buildCache
	if err := json.Unmarshal(b, &c); err != nil {
		// Если кэш битый — начинаем с нуля.
		return newBuildCache(), nil
	}
	if c.Version != 1 {
		return newBuildCache(), nil
	}
	if c.Godsl == nil {
		c.Godsl = map[string]cacheEntry{}
	}
	if c.Files == nil {
		c.Files = map[string]cacheEntry{}
	}
	return c, nil
}

func saveBuildCache(buildDir string, cache buildCache) error {
	path := filepath.Join(buildDir, cacheFileName)
	tmp := path + ".tmp"

	b, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}

func fileSHA256Hex(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения файла %s: %w", path, err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func cleanupEmptyDirs(root, dir string) error {
	root = filepath.Clean(root)
	dir = filepath.Clean(dir)

	for {
		if dir == root || dir == "." || dir == string(filepath.Separator) {
			return nil
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		// Не удаляем директорию, если там лежит кэш.
		if len(entries) == 0 {
			if err := os.Remove(dir); err != nil {
				return nil
			}
			dir = filepath.Dir(dir)
			continue
		}
		return nil
	}
}

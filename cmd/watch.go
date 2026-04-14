package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

// godslFileWatcher отслеживает изменения .godsl файлов через polling
type godslFileWatcher struct {
	rootDir    string
	prevMtimes map[string]time.Time
}

func newGodslFileWatcher(projectPath string) (*godslFileWatcher, error) {
	rootDir := projectPath
	if rootDir == "" {
		var err error
		rootDir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}
	w := &godslFileWatcher{
		rootDir:    absRoot,
		prevMtimes: make(map[string]time.Time),
	}
	// Инициализируем начальное состояние без репортинга
	w.prevMtimes = w.scanFiles()
	return w, nil
}

// scanFiles обходит rootDir и возвращает mtime всех .godsl файлов
func (w *godslFileWatcher) scanFiles() map[string]time.Time {
	mtimes := make(map[string]time.Time)
	_ = filepath.Walk(w.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && info.Name() == "build" {
			return filepath.SkipDir
		}
		if !info.IsDir() && filepath.Ext(path) == ".godsl" {
			mtimes[path] = info.ModTime()
		}
		return nil
	})
	return mtimes
}

// hasChanges проверяет, изменились ли .godsl файлы с последней проверки.
// При обнаружении изменений обновляет внутреннее состояние.
func (w *godslFileWatcher) hasChanges() bool {
	current := w.scanFiles()
	changed := false

	for path, mtime := range current {
		if prev, ok := w.prevMtimes[path]; !ok || !mtime.Equal(prev) {
			fmt.Printf("  ~ %s\n", filepath.Base(path))
			changed = true
		}
	}
	for path := range w.prevMtimes {
		if _, ok := current[path]; !ok {
			fmt.Printf("  - %s (удалён)\n", filepath.Base(path))
			changed = true
		}
	}

	if changed {
		w.prevMtimes = current
	}
	return changed
}

// watchGenerate запускает generate в режиме наблюдения:
// автоматически перегенерирует при изменении .godsl файлов
func watchGenerate(projectPath string, clean bool) {
	w, err := newGodslFileWatcher(projectPath)
	if err != nil {
		fmt.Printf("Ошибка инициализации watcher: %v\n", err)
		return
	}

	fmt.Println("[watch] Начальная генерация...")
	if _, err := generateProject(projectPath, "", GenerateOptions{Clean: clean}); err != nil {
		fmt.Printf("Ошибка генерации: %v\n", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	fmt.Println("[watch] Наблюдение за .godsl файлами... (Ctrl+C для остановки)")
	for {
		select {
		case <-sigCh:
			fmt.Println("\n[watch] Остановка.")
			return
		case <-ticker.C:
			if w.hasChanges() {
				fmt.Println("[watch] Изменения обнаружены, регенерация...")
				if _, err := generateProject(projectPath, "", GenerateOptions{}); err != nil {
					fmt.Printf("Ошибка генерации: %v\n", err)
				}
			}
		}
	}
}

// watchRun запускает программу и перезапускает её при каждом изменении .godsl файлов.
// Компилирует через go build во временный бинарник для чистого управления процессом.
func watchRun(projectPath string, clean bool) {
	w, err := newGodslFileWatcher(projectPath)
	if err != nil {
		fmt.Printf("Ошибка инициализации watcher: %v\n", err)
		return
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	var currentProc *os.Process
	var tmpBin string

	// killCurrent останавливает текущий запущенный процесс
	killCurrent := func() {
		if currentProc != nil {
			_ = currentProc.Kill()
			_, _ = currentProc.Wait()
			currentProc = nil
		}
		if tmpBin != "" {
			_ = os.Remove(tmpBin)
			tmpBin = ""
		}
	}

	// startProgram транспилирует, собирает и запускает программу
	startProgram := func(firstRun bool) {
		killCurrent()

		opts := GenerateOptions{}
		if firstRun {
			opts.Clean = clean
		}

		buildDir, err := generateProject(projectPath, "", opts)
		if err != nil {
			fmt.Printf("[watch] Ошибка генерации: %v\n", err)
			return
		}
		execDir := goBuildExecDir(buildDir, projectPath)

		// Компилируем в бинарник — kill() на него работает напрямую
		tmpBin = filepath.Join(execDir, ".godsl_watch_bin")
		buildCmd := exec.Command("go", "build", "-o", tmpBin, ".")
		buildCmd.Dir = execDir
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if err := buildCmd.Run(); err != nil {
			fmt.Printf("[watch] Ошибка компиляции: %v\n", err)
			tmpBin = ""
			return
		}

		runCmd := exec.Command(tmpBin)
		runCmd.Dir = execDir
		runCmd.Stdout = os.Stdout
		runCmd.Stderr = os.Stderr
		runCmd.Stdin = os.Stdin
		if err := runCmd.Start(); err != nil {
			fmt.Printf("[watch] Ошибка запуска: %v\n", err)
			return
		}
		currentProc = runCmd.Process
		fmt.Printf("[watch] Запущен (pid %d)\n", currentProc.Pid)
	}

	startProgram(true)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	fmt.Println("[watch] Наблюдение за .godsl файлами... (Ctrl+C для остановки)")
	for {
		select {
		case <-sigCh:
			fmt.Println("\n[watch] Остановка.")
			killCurrent()
			return
		case <-ticker.C:
			if w.hasChanges() {
				fmt.Println("[watch] Изменения — перезапуск...")
				startProgram(false)
			}
		}
	}
}

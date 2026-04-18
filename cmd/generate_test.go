package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── goBuildExecDir ───────────────────────────────────────────────────────────

func TestGoBuildExecDir_EmptyProjectPath(t *testing.T) {
	buildDir := "/some/build/dir"
	got := goBuildExecDir(buildDir, "")
	if got != buildDir {
		t.Errorf("goBuildExecDir(%q, \"\") = %q, want %q", buildDir, got, buildDir)
	}
}

func TestGoBuildExecDir_WhitespaceProjectPath(t *testing.T) {
	buildDir := "/some/build/dir"
	got := goBuildExecDir(buildDir, "   ")
	if got != buildDir {
		t.Errorf("goBuildExecDir(%q, \"   \") = %q, want %q", buildDir, got, buildDir)
	}
}

func TestGoBuildExecDir_RelativeProjectPath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	buildDir := filepath.Join(cwd, "build")

	// projectPath is a subdirectory of cwd
	subdir := filepath.Join(cwd, "examples")
	got := goBuildExecDir(buildDir, subdir)
	want := filepath.Join(buildDir, "examples")
	if got != want {
		t.Errorf("goBuildExecDir: got %q, want %q", got, want)
	}
}

func TestGoBuildExecDir_ProjectPathOutsideCwd_FallsBackToBuildDir(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	buildDir := filepath.Join(cwd, "build")

	// Project path that escapes cwd → should return buildDir
	outsidePath := filepath.Join(cwd, "..", "..", "outside")
	got := goBuildExecDir(buildDir, outsidePath)
	if got != buildDir {
		t.Errorf("expected fallback to buildDir, got %q", got)
	}
}

// ─── planProjectTasks ─────────────────────────────────────────────────────────

func TestPlanProjectTasks_NewGodslFile(t *testing.T) {
	srcDir := t.TempDir()
	buildDir := t.TempDir()

	// Create a .godsl file in srcDir
	if err := os.WriteFile(filepath.Join(srcDir, "main.godsl"), []byte(`package main

func main() {}
`), 0644); err != nil {
		t.Fatal(err)
	}

	cache := newBuildCache()
	tasks, _, _, _, _, nextCache, err := planProjectTasks(srcDir, srcDir, buildDir, cache)
	if err != nil {
		t.Fatalf("planProjectTasks error: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
	if len(nextCache.Godsl) != 1 {
		t.Errorf("expected 1 cache entry, got %d", len(nextCache.Godsl))
	}
}

func TestPlanProjectTasks_CopiesNonGodslFiles(t *testing.T) {
	srcDir := t.TempDir()
	buildDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module test\ngo 1.22\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cache := newBuildCache()
	_, copyTasks, _, _, _, _, err := planProjectTasks(srcDir, srcDir, buildDir, cache)
	if err != nil {
		t.Fatalf("planProjectTasks error: %v", err)
	}
	if len(copyTasks) != 1 {
		t.Errorf("expected 1 copy task for go.mod, got %d", len(copyTasks))
	}
}

func TestPlanProjectTasks_IncrementalBuild_CachesUnchangedFiles(t *testing.T) {
	srcDir := t.TempDir()
	buildDir := t.TempDir()

	godslPath := filepath.Join(srcDir, "main.godsl")
	content := []byte("package main\nfunc main() {}\n")
	if err := os.WriteFile(godslPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	// First pass — should schedule transpilation
	cache := newBuildCache()
	tasks1, _, _, _, _, nextCache, err := planProjectTasks(srcDir, srcDir, buildDir, cache)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks1) != 1 {
		t.Errorf("first pass: expected 1 task, got %d", len(tasks1))
	}

	// Second pass with the updated cache — file unchanged, should be cached
	tasks2, _, _, cachedGodsl, _, _, err := planProjectTasks(srcDir, srcDir, buildDir, nextCache)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks2) != 0 {
		t.Errorf("second pass: expected 0 tasks (cached), got %d", len(tasks2))
	}
	if len(cachedGodsl) != 1 {
		t.Errorf("second pass: expected 1 cached godsl entry, got %d", len(cachedGodsl))
	}
}

func TestPlanProjectTasks_DetectsDeletedFiles(t *testing.T) {
	srcDir := t.TempDir()
	buildDir := t.TempDir()

	// Prime cache with a file that no longer exists
	cache := newBuildCache()
	cache.Godsl["old.godsl"] = cacheEntry{TargetRel: "old.go"}

	_, _, deletions, _, _, _, err := planProjectTasks(srcDir, srcDir, buildDir, cache)
	if err != nil {
		t.Fatal(err)
	}
	if len(deletions) != 1 {
		t.Errorf("expected 1 deletion (for old.go), got %d", len(deletions))
	}
}

func TestPlanProjectTasks_SkipsBuildDir(t *testing.T) {
	srcDir := t.TempDir()
	buildDir := filepath.Join(srcDir, "build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Place a .godsl file inside build/ — should be skipped
	if err := os.WriteFile(filepath.Join(buildDir, "skip.godsl"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	cache := newBuildCache()
	tasks, _, _, _, _, _, err := planProjectTasks(srcDir, srcDir, buildDir, cache)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks (build dir skipped), got %d", len(tasks))
	}
}

// ─── generateProject integration ─────────────────────────────────────────────

func TestGenerateProject_BasicTranspilation(t *testing.T) {
	srcDir := t.TempDir()

	// Write a minimal .godsl project
	gomod := "module testapp\n\ngo 1.22\n"
	godsl := `package main

import "fmt"

func main() {
	try {
		@errcheck
		_, err := riskyOp()
	} catch {
		fmt.Println(err)
	}
}

func riskyOp() (int, error) { return 0, nil }
`
	mustWriteFile(t, filepath.Join(srcDir, "go.mod"), gomod)
	mustWriteFile(t, filepath.Join(srcDir, "main.godsl"), godsl)

	buildDir := t.TempDir()

	// generateProject uses os.Getwd() as relBase, so we chdir to srcDir
	origWd := mustChdir(t, srcDir)
	defer os.Chdir(origWd)

	resultDir, err := generateProject("", buildDir, GenerateOptions{})
	if err != nil {
		t.Fatalf("generateProject error: %v", err)
	}
	if resultDir != buildDir {
		t.Errorf("expected resultDir=%q, got %q", buildDir, resultDir)
	}

	// main.go must be created
	mainGo := filepath.Join(buildDir, "main.go")
	data, err := os.ReadFile(mainGo)
	if err != nil {
		t.Fatalf("main.go not created: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "if err != nil") {
		t.Errorf("transpiled output missing 'if err != nil'\n\nContent:\n%s", content)
	}
	if strings.Contains(content, "try {") {
		t.Errorf("transpiled output still contains 'try {'\n\nContent:\n%s", content)
	}
	if strings.Contains(content, "@errcheck") {
		t.Errorf("transpiled output still contains '@errcheck'\n\nContent:\n%s", content)
	}
}

func TestGenerateProject_CopiesGoMod(t *testing.T) {
	srcDir := t.TempDir()
	gomod := "module testapp\n\ngo 1.22\n"
	mustWriteFile(t, filepath.Join(srcDir, "go.mod"), gomod)
	mustWriteFile(t, filepath.Join(srcDir, "main.godsl"), "package main\nfunc main(){}\n")

	buildDir := t.TempDir()
	origWd := mustChdir(t, srcDir)
	defer os.Chdir(origWd)

	if _, err := generateProject("", buildDir, GenerateOptions{}); err != nil {
		t.Fatalf("generateProject error: %v", err)
	}

	goModDst := filepath.Join(buildDir, "go.mod")
	if _, err := os.Stat(goModDst); err != nil {
		t.Errorf("go.mod was not copied to build dir: %v", err)
	}
}

func TestGenerateProject_IncrementalBuild_SecondRun_Cached(t *testing.T) {
	srcDir := t.TempDir()
	mustWriteFile(t, filepath.Join(srcDir, "go.mod"), "module testapp\ngo 1.22\n")
	mustWriteFile(t, filepath.Join(srcDir, "main.godsl"), "package main\nfunc main(){}\n")

	buildDir := t.TempDir()
	origWd := mustChdir(t, srcDir)
	defer os.Chdir(origWd)

	// First run
	if _, err := generateProject("", buildDir, GenerateOptions{}); err != nil {
		t.Fatalf("first run error: %v", err)
	}
	// Second run without changes — should succeed (uses cache)
	if _, err := generateProject("", buildDir, GenerateOptions{}); err != nil {
		t.Fatalf("second run (incremental) error: %v", err)
	}
}

func TestGenerateProject_CleanBuild_RegeneatesEverything(t *testing.T) {
	srcDir := t.TempDir()
	mustWriteFile(t, filepath.Join(srcDir, "go.mod"), "module testapp\ngo 1.22\n")
	mustWriteFile(t, filepath.Join(srcDir, "main.godsl"), "package main\nfunc main(){}\n")

	buildDir := t.TempDir()
	origWd := mustChdir(t, srcDir)
	defer os.Chdir(origWd)

	if _, err := generateProject("", buildDir, GenerateOptions{}); err != nil {
		t.Fatalf("first run error: %v", err)
	}
	// Clean rebuild should not error
	if _, err := generateProject("", buildDir, GenerateOptions{Clean: true}); err != nil {
		t.Fatalf("clean build error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(buildDir, "main.go")); err != nil {
		t.Errorf("main.go missing after clean rebuild: %v", err)
	}
}

func TestGenerateProject_NonExistentPath_ReturnsError(t *testing.T) {
	_, err := generateProject("/this/path/does/not/exist/at/all", "", GenerateOptions{})
	if err == nil {
		t.Error("expected error for non-existent project path, got nil")
	}
}

func TestGenerateProject_MultipleGodslFiles(t *testing.T) {
	srcDir := t.TempDir()
	mustWriteFile(t, filepath.Join(srcDir, "go.mod"), "module testapp\ngo 1.22\n")
	mustWriteFile(t, filepath.Join(srcDir, "main.godsl"), "package main\nfunc main(){}\n")
	mustWriteFile(t, filepath.Join(srcDir, "util.godsl"), "package main\nfunc helper() {}\n")

	buildDir := t.TempDir()
	origWd := mustChdir(t, srcDir)
	defer os.Chdir(origWd)

	if _, err := generateProject("", buildDir, GenerateOptions{}); err != nil {
		t.Fatalf("generateProject error: %v", err)
	}
	for _, name := range []string{"main.go", "util.go"} {
		if _, err := os.Stat(filepath.Join(buildDir, name)); err != nil {
			t.Errorf("expected %s in build dir: %v", name, err)
		}
	}
}

func TestGenerateProject_PreservesSubdirectoryStructure(t *testing.T) {
	srcDir := t.TempDir()
	subDir := filepath.Join(srcDir, "internal", "util")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(srcDir, "go.mod"), "module testapp\ngo 1.22\n")
	mustWriteFile(t, filepath.Join(srcDir, "main.godsl"), "package main\nfunc main(){}\n")
	mustWriteFile(t, filepath.Join(subDir, "util.godsl"), "package util\nfunc Foo() {}\n")

	buildDir := t.TempDir()
	origWd := mustChdir(t, srcDir)
	defer os.Chdir(origWd)

	if _, err := generateProject("", buildDir, GenerateOptions{}); err != nil {
		t.Fatalf("generateProject error: %v", err)
	}

	expectedPath := filepath.Join(buildDir, "internal", "util", "util.go")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("expected %s in build: %v", expectedPath, err)
	}
}

// ─── build cache persistence ──────────────────────────────────────────────────

func TestBuildCache_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	cache := newBuildCache()
	cache.Godsl["main.godsl"] = cacheEntry{TargetRel: "main.go", Size: 100, Hash: "abc123"}
	cache.Files["go.mod"] = cacheEntry{TargetRel: "go.mod", Size: 50}

	if err := saveBuildCache(dir, cache); err != nil {
		t.Fatalf("saveBuildCache error: %v", err)
	}

	loaded, err := loadBuildCache(dir)
	if err != nil {
		t.Fatalf("loadBuildCache error: %v", err)
	}
	if len(loaded.Godsl) != 1 {
		t.Errorf("expected 1 godsl entry, got %d", len(loaded.Godsl))
	}
	entry, ok := loaded.Godsl["main.godsl"]
	if !ok {
		t.Fatal("expected 'main.godsl' in cache")
	}
	if entry.Hash != "abc123" {
		t.Errorf("expected hash='abc123', got %q", entry.Hash)
	}
}

func TestBuildCache_LoadMissingFile_ReturnsEmptyCache(t *testing.T) {
	dir := t.TempDir()
	cache, err := loadBuildCache(dir)
	if err != nil {
		t.Fatalf("unexpected error for missing cache: %v", err)
	}
	if len(cache.Godsl) != 0 || len(cache.Files) != 0 {
		t.Error("expected empty cache for missing file")
	}
}

func TestBuildCache_LoadCorruptFile_ReturnsEmptyCache(t *testing.T) {
	dir := t.TempDir()
	cacheFile := filepath.Join(dir, cacheFileName)
	if err := os.WriteFile(cacheFile, []byte("{ this is not valid json"), 0644); err != nil {
		t.Fatal(err)
	}
	cache, err := loadBuildCache(dir)
	if err != nil {
		t.Fatalf("expected no error for corrupt cache, got: %v", err)
	}
	if len(cache.Godsl) != 0 {
		t.Error("expected empty cache on corrupt file")
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}

func mustChdir(t *testing.T, dir string) string {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir to %s: %v", dir, err)
	}
	return orig
}

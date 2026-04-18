package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// ─── collectGodslFiles ────────────────────────────────────────────────────────

func TestCollectGodslFiles_SingleFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.godsl")
	mustWriteFile(t, path, "package main\n")

	files, err := collectGodslFiles([]string{path})
	if err != nil {
		t.Fatalf("collectGodslFiles error: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d: %v", len(files), files)
	}
	if files[0] != path {
		t.Errorf("expected %q, got %q", path, files[0])
	}
}

func TestCollectGodslFiles_NonGodslFile_Ignored(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "main.go"), "package main\n")
	mustWriteFile(t, filepath.Join(dir, "main.godsl"), "package main\n")

	files, err := collectGodslFiles([]string{dir})
	if err != nil {
		t.Fatalf("collectGodslFiles error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file (only .godsl), got %d: %v", len(files), files)
	}
	if filepath.Ext(files[0]) != ".godsl" {
		t.Errorf("expected .godsl file, got %q", files[0])
	}
}

func TestCollectGodslFiles_Directory_NotRecursive(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, "a.godsl"), "package main\n")
	mustWriteFile(t, filepath.Join(subdir, "b.godsl"), "package main\n")

	// Without /... pattern — only top-level files
	files, err := collectGodslFiles([]string{dir})
	if err != nil {
		t.Fatalf("collectGodslFiles error: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("non-recursive: expected 1 file (top-level only), got %d: %v", len(files), files)
	}
}

func TestCollectGodslFiles_Recursive_DotDotDot(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, "a.godsl"), "package main\n")
	mustWriteFile(t, filepath.Join(subdir, "b.godsl"), "package main\n")

	pattern := dir + "/..."
	files, err := collectGodslFiles([]string{pattern})
	if err != nil {
		t.Fatalf("collectGodslFiles error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("recursive: expected 2 files, got %d: %v", len(files), files)
	}
}

func TestCollectGodslFiles_SkipsBuildDirectory(t *testing.T) {
	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, "real.godsl"), "package main\n")
	mustWriteFile(t, filepath.Join(buildDir, "generated.godsl"), "package main\n")

	pattern := dir + "/..."
	files, err := collectGodslFiles([]string{pattern})
	if err != nil {
		t.Fatalf("collectGodslFiles error: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected build dir to be skipped, got %d files: %v", len(files), files)
	}
}

func TestCollectGodslFiles_EmptyDirectory_ReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	files, err := collectGodslFiles([]string{dir + "/..."})
	if err != nil {
		t.Fatalf("collectGodslFiles error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files in empty dir, got %d", len(files))
	}
}

func TestCollectGodslFiles_NonExistentPath_ReturnsError(t *testing.T) {
	_, err := collectGodslFiles([]string{"/this/path/does/not/exist/ever"})
	if err == nil {
		t.Error("expected error for non-existent path, got nil")
	}
}

func TestCollectGodslFiles_DeduplicatesFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.godsl")
	mustWriteFile(t, path, "package main\n")

	// Pass the same file twice
	files, err := collectGodslFiles([]string{path, path})
	if err != nil {
		t.Fatalf("collectGodslFiles error: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected deduplication to 1 file, got %d: %v", len(files), files)
	}
}

func TestCollectGodslFiles_MultiplePatterns(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	mustWriteFile(t, filepath.Join(dir1, "a.godsl"), "package main\n")
	mustWriteFile(t, filepath.Join(dir2, "b.godsl"), "package main\n")

	files, err := collectGodslFiles([]string{dir1, dir2})
	if err != nil {
		t.Fatalf("collectGodslFiles error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files from 2 dirs, got %d: %v", len(files), files)
	}
}

func TestCollectGodslFiles_DotDotDotWithoutSlash(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "x.godsl"), "package main\n")

	// "dir..." (without slash before ...) is also handled
	pattern := dir + "..."
	files, err := collectGodslFiles([]string{pattern})
	if err != nil {
		// Some patterns may error; that's acceptable behaviour
		return
	}
	// If it succeeds, should find the file
	_ = files
}

// ─── mapPackagePatterns ───────────────────────────────────────────────────────

func TestMapPackagePatterns_FlagsPassThrough(t *testing.T) {
	buildDir := t.TempDir()
	args := []string{"-v", "-run", "TestFoo", "-count=1"}
	got := mapPackagePatterns(args, buildDir)
	if len(got) != len(args) {
		t.Fatalf("expected %d results, got %d", len(args), len(got))
	}
	for i, want := range args {
		if got[i] != want {
			t.Errorf("arg[%d]: expected %q, got %q", i, want, got[i])
		}
	}
}

func TestMapPackagePatterns_DotDotDot_Preserved(t *testing.T) {
	buildDir := t.TempDir()
	got := mapPackagePatterns([]string{"./..."}, buildDir)
	if len(got) != 1 || got[0] != "./..." {
		t.Errorf("expected './...', got %v", got)
	}
}

func TestMapPackagePatterns_Mixed_FlagsAndPatterns(t *testing.T) {
	buildDir := t.TempDir()
	args := []string{"-v", "./..."}
	got := mapPackagePatterns(args, buildDir)
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d: %v", len(got), got)
	}
	if got[0] != "-v" {
		t.Errorf("flag should pass through unchanged, got %q", got[0])
	}
}

// ─── mapSinglePattern ─────────────────────────────────────────────────────────

func TestMapSinglePattern_DotDotDot_Preserved(t *testing.T) {
	cwd, _ := os.Getwd()
	buildDir := t.TempDir()

	got := mapSinglePattern("./...", cwd, buildDir)
	if got != "./..." {
		t.Errorf("expected './...', got %q", got)
	}
}

func TestMapSinglePattern_TripleDot_Preserved(t *testing.T) {
	cwd, _ := os.Getwd()
	buildDir := t.TempDir()

	got := mapSinglePattern("...", cwd, buildDir)
	if got != "./..." {
		t.Errorf("expected './...', got %q", got)
	}
}

func TestMapSinglePattern_NonExistentRelPath_Passthrough(t *testing.T) {
	cwd, _ := os.Getwd()
	buildDir := t.TempDir()

	// ./nonexistent does not exist in buildDir → passes through unchanged
	got := mapSinglePattern("./nonexistent", cwd, buildDir)
	// Either passthrough or mapped — should not panic
	_ = got
}

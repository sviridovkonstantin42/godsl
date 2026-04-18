package transpiler_test

import (
	"strings"
	"testing"

	"github.com/sviridovkonstantin42/godsl/internal/transpiler"
)

// ─── FormatFile ───────────────────────────────────────────────────────────────

func TestFormatFile_BasicCode(t *testing.T) {
	src := `package main

func foo() int {
return 42
}
`
	out, err := transpiler.FormatFile(src)
	if err != nil {
		t.Fatalf("FormatFile returned error: %v", err)
	}
	if out == "" {
		t.Fatal("FormatFile returned empty string")
	}
	if !strings.Contains(out, "return 42") {
		t.Errorf("formatted output missing 'return 42'\n\nOutput:\n%s", out)
	}
}

func TestFormatFile_TryCatch_Preserved(t *testing.T) {
	src := `package main

func foo() error {
try {
@errcheck
_, err := bar()
} catch {
return err
}
return nil
}

func bar() (int, error) { return 0, nil }
`
	out, err := transpiler.FormatFile(src)
	if err != nil {
		t.Fatalf("FormatFile returned error: %v", err)
	}
	// FormatFile should NOT transpile — try/catch must remain in output
	if !strings.Contains(out, "try") {
		t.Errorf("FormatFile should preserve 'try' keyword, but it's missing\n\nOutput:\n%s", out)
	}
	if !strings.Contains(out, "catch") {
		t.Errorf("FormatFile should preserve 'catch' keyword, but it's missing\n\nOutput:\n%s", out)
	}
}

func TestFormatFile_Finally_Preserved(t *testing.T) {
	src := `package main

import "fmt"

func foo() {
try {
@errcheck
_, err := bar()
} catch {
fmt.Println(err)
} finally {
fmt.Println("cleanup")
}
}

func bar() (int, error) { return 0, nil }
`
	out, err := transpiler.FormatFile(src)
	if err != nil {
		t.Fatalf("FormatFile returned error: %v", err)
	}
	if !strings.Contains(out, "finally") {
		t.Errorf("FormatFile should preserve 'finally' keyword\n\nOutput:\n%s", out)
	}
}

func TestFormatFile_Throw_Preserved(t *testing.T) {
	src := `package main

import "errors"

func foo() error {
throw errors.New("oops")
}
`
	out, err := transpiler.FormatFile(src)
	if err != nil {
		t.Fatalf("FormatFile returned error: %v", err)
	}
	if !strings.Contains(out, "throw") {
		t.Errorf("FormatFile should preserve 'throw' keyword\n\nOutput:\n%s", out)
	}
}

func TestFormatFile_Must_Preserved(t *testing.T) {
	src := `package main

func foo() {
must validate()
}

func validate() error { return nil }
`
	out, err := transpiler.FormatFile(src)
	if err != nil {
		t.Fatalf("FormatFile returned error: %v", err)
	}
	if !strings.Contains(out, "must") {
		t.Errorf("FormatFile should preserve 'must' keyword\n\nOutput:\n%s", out)
	}
}

func TestFormatFile_QuestionOp_Preserved(t *testing.T) {
	src := `package main

func foo() error {
a := bar()?
_ = a
return nil
}

func bar() (int, error) { return 0, nil }
`
	out, err := transpiler.FormatFile(src)
	if err != nil {
		t.Fatalf("FormatFile returned error: %v", err)
	}
	if !strings.Contains(out, "?") {
		t.Errorf("FormatFile should preserve '?' operator\n\nOutput:\n%s", out)
	}
}

func TestFormatFile_Idempotent(t *testing.T) {
	src := `package main

func foo() error {
	try {
		@errcheck
		_, err := bar()
	} catch {
		return err
	}
	return nil
}

func bar() (int, error) { return 0, nil }
`
	out1, err := transpiler.FormatFile(src)
	if err != nil {
		t.Fatalf("first FormatFile error: %v", err)
	}
	out2, err := transpiler.FormatFile(out1)
	if err != nil {
		t.Fatalf("second FormatFile error: %v", err)
	}
	if out1 != out2 {
		t.Errorf("FormatFile is not idempotent\n\nFirst:\n%s\n\nSecond:\n%s", out1, out2)
	}
}

func TestFormatFile_InvalidSyntax_ReturnsError(t *testing.T) {
	src := `this is not valid godsl @@@@`
	_, err := transpiler.FormatFile(src)
	if err == nil {
		t.Error("expected FormatFile to return an error for invalid input, got nil")
	}
}

func TestFormatFile_EmptyPackage(t *testing.T) {
	src := `package main
`
	out, err := transpiler.FormatFile(src)
	if err != nil {
		t.Fatalf("FormatFile returned error for minimal package: %v", err)
	}
	if !strings.Contains(out, "package main") {
		t.Errorf("expected output to contain 'package main'\n\nOutput:\n%s", out)
	}
}

func TestFormatFile_NormalizesIndentation(t *testing.T) {
	// Poorly indented input should be canonically re-indented
	src := `package main
func foo() int {
return 42
}
`
	out, err := transpiler.FormatFile(src)
	if err != nil {
		t.Fatalf("FormatFile error: %v", err)
	}
	// After formatting, return statement should be indented with a tab
	if !strings.Contains(out, "\treturn 42") {
		t.Errorf("expected return to be tab-indented\n\nOutput:\n%s", out)
	}
}

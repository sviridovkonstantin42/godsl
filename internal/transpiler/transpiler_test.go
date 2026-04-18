package transpiler_test

import (
	goparser "go/parser"
	gotoken "go/token"
	"strings"
	"testing"

	"github.com/sviridovkonstantin42/godsl/internal/transpiler"
)

// Re-export for direct call in TestTranspileFile_Must_ReassignmentForm
var _ = transpiler.TranspileFile

// ─── helpers ──────────────────────────────────────────────────────────────────

// assertValidGo verifies that code parses as syntactically valid Go.
func assertValidGo(t *testing.T, code string) {
	t.Helper()
	fset := gotoken.NewFileSet()
	if _, err := goparser.ParseFile(fset, "", code, 0); err != nil {
		t.Errorf("transpiled output is not valid Go:\n%v\n\nOutput:\n%s", err, code)
	}
}

func assertContains(t *testing.T, code, want string) {
	t.Helper()
	if !strings.Contains(code, want) {
		t.Errorf("expected output to contain %q\n\nActual:\n%s", want, code)
	}
}

func assertNotContains(t *testing.T, code, want string) {
	t.Helper()
	if strings.Contains(code, want) {
		t.Errorf("expected output NOT to contain %q\n\nActual:\n%s", want, code)
	}
}

func transpileOK(t *testing.T, src string) string {
	t.Helper()
	out, err := transpiler.TranspileFile(src)
	if err != nil {
		t.Fatalf("TranspileFile returned unexpected error: %v", err)
	}
	return out
}

// ─── plain Go passthrough ─────────────────────────────────────────────────────

func TestTranspileFile_PlainGo_Passthrough(t *testing.T) {
	src := `package main

import "fmt"

func main() {
	fmt.Println("hello, world")
}
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, `fmt.Println("hello, world")`)
}

func TestTranspileFile_PlainGo_WithIfAndFor(t *testing.T) {
	src := `package main

func foo(x int) int {
	if x > 0 {
		return x
	}
	result := 0
	for i := 0; i < x; i++ {
		result += i
	}
	return result
}
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "if x > 0")
	assertContains(t, out, "for i := 0")
}

// ─── try / catch with @errcheck ───────────────────────────────────────────────

func TestTranspileFile_TryCatch_NewAnnotation(t *testing.T) {
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
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "if err != nil")
	assertContains(t, out, "return err")
	assertNotContains(t, out, "try {")
	assertNotContains(t, out, "} catch")
	assertNotContains(t, out, "@errcheck")
}

func TestTranspileFile_TryCatch_CommentAnnotation_LineAbove(t *testing.T) {
	src := `package main

func foo() error {
	try {
		//@errcheck
		_, err := bar()
	} catch {
		return err
	}
	return nil
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "if err != nil")
	assertContains(t, out, "return err")
	assertNotContains(t, out, "try {")
	assertNotContains(t, out, "@errcheck")
}

func TestTranspileFile_TryCatch_CommentAnnotation_SameLine(t *testing.T) {
	src := `package main

func foo() error {
	try {
		_, err := bar() //@errcheck
	} catch {
		return err
	}
	return nil
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "if err != nil")
	assertNotContains(t, out, "try {")
}

func TestTranspileFile_TryCatch_MultipleErrChecks(t *testing.T) {
	src := `package main

func foo() error {
	try {
		@errcheck
		a, err := bar()
		_ = a

		@errcheck
		b, err := bar()
		_ = b
	} catch {
		return err
	}
	return nil
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	// Two separate if-blocks should be generated
	count := strings.Count(out, "if err != nil")
	if count != 2 {
		t.Errorf("expected 2 'if err != nil' blocks, got %d\n\nOutput:\n%s", count, out)
	}
	assertNotContains(t, out, "try {")
}

func TestTranspileFile_TryCatch_CodeBetweenErrChecks(t *testing.T) {
	src := `package main

import "fmt"

func foo() error {
	try {
		@errcheck
		a, err := bar()
		fmt.Println("step 1", a)

		@errcheck
		b, err := bar()
		fmt.Println("step 2", b)
	} catch {
		return err
	}
	return nil
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, `fmt.Println("step 1"`)
	assertContains(t, out, `fmt.Println("step 2"`)
	if strings.Count(out, "if err != nil") != 2 {
		t.Errorf("expected 2 error checks\n\nOutput:\n%s", out)
	}
}

// ─── try / catch without @errcheck (plain body) ───────────────────────────────

func TestTranspileFile_TryCatch_NoCatch(t *testing.T) {
	src := `package main

func foo() error {
	try {
		@errcheck
		_, err := bar()
	}
	return nil
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	// No catches → generated: if err != nil { return err }
	assertContains(t, out, "if err != nil")
	assertContains(t, out, "return err")
}

// ─── typed catch ──────────────────────────────────────────────────────────────

func TestTranspileFile_TryCatch_TypedCatch_Single(t *testing.T) {
	src := `package main

import "errors"

type MyError struct{}

func (e MyError) Error() string { return "my error" }

func foo() error {
	try {
		@errcheck
		_, err := bar()
	} catch(MyError) {
		return errors.New("caught MyError")
	}
	return nil
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "err.(MyError)")
	assertContains(t, out, `errors.New("caught MyError")`)
	assertNotContains(t, out, "} catch")
}

func TestTranspileFile_TryCatch_TypedCatch_WithVar(t *testing.T) {
	src := `package main

import "fmt"

type MyError struct{ Msg string }

func (e MyError) Error() string { return e.Msg }

func foo() error {
	try {
		@errcheck
		_, err := bar()
	} catch(e MyError) {
		fmt.Println(e.Msg)
		return e
	}
	return nil
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	// Type assertion with variable binding: e, ok := err.(MyError)
	assertContains(t, out, "err.(MyError)")
	assertContains(t, out, "e.Msg")
}

func TestTranspileFile_TryCatch_TypedCatch_Qualified(t *testing.T) {
	src := `package main

import "os"

func foo() error {
	try {
		@errcheck
		_, err := bar()
	} catch(os.PathError) {
		return nil
	}
	return nil
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "os.PathError")
}

func TestTranspileFile_TryCatch_MultiTypeCatch_Pipe(t *testing.T) {
	src := `package main

type ErrA struct{}
type ErrB struct{}

func (e ErrA) Error() string { return "a" }
func (e ErrB) Error() string { return "b" }

func foo() error {
	try {
		@errcheck
		_, err := bar()
	} catch(ErrA | ErrB) {
		return err
	}
	return nil
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	// Multi-type → IIFE with individual type assertions
	assertContains(t, out, "ErrA")
	assertContains(t, out, "ErrB")
	assertNotContains(t, out, "} catch")
}

func TestTranspileFile_TryCatch_MultiTypeCatch_WithVar(t *testing.T) {
	src := `package main

import "fmt"

type ErrA struct{}
type ErrB struct{}

func (e ErrA) Error() string { return "a" }
func (e ErrB) Error() string { return "b" }

func foo() error {
	try {
		@errcheck
		_, err := bar()
	} catch(e ErrA | ErrB) {
		fmt.Println(e)
		return e
	}
	return nil
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "ErrA")
	assertContains(t, out, "ErrB")
}

func TestTranspileFile_TryCatch_TypedThenCatchAll(t *testing.T) {
	src := `package main

import "errors"

type MyError struct{}

func (e MyError) Error() string { return "my" }

func foo() error {
	try {
		@errcheck
		_, err := bar()
	} catch(MyError) {
		return errors.New("typed")
	} catch {
		return err
	}
	return nil
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "err.(MyError)")
	assertContains(t, out, `errors.New("typed")`)
}

// ─── finally ──────────────────────────────────────────────────────────────────

func TestTranspileFile_Finally_NoReturnInCatch(t *testing.T) {
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
	out := transpileOK(t, src)
	assertValidGo(t, out)
	// Finally body must appear in output
	assertContains(t, out, `fmt.Println("cleanup")`)
	// IIFE pattern should be present
	assertContains(t, out, "func() bool")
	// No _godslRet capture (catch has no return)
	assertNotContains(t, out, "_godslRet")
	assertNotContains(t, out, "finally")
}

func TestTranspileFile_Finally_WithReturnInCatch(t *testing.T) {
	src := `package main

import "fmt"

func foo() {
	try {
		@errcheck
		_, err := bar()
	} catch {
		fmt.Println(err)
		return
	} finally {
		fmt.Println("cleanup")
	}
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, `fmt.Println("cleanup")`)
	// Return in catch → IIFE with _godslRet capture
	assertContains(t, out, "_godslRet")
	assertContains(t, out, "if _godslRet")
	assertNotContains(t, out, "finally")
}

func TestTranspileFile_Finally_FinallyBodyAlwaysRuns(t *testing.T) {
	src := `package main

import "fmt"

func foo() {
	x := 0
	try {
		@errcheck
		_, err := bar()
	} catch {
		fmt.Println(err)
	} finally {
		x++
	}
	_ = x
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "x++")
}

// ─── ? operator ───────────────────────────────────────────────────────────────

func TestTranspileFile_QuestionOp_Assignment(t *testing.T) {
	src := `package main

func foo() error {
	a := bar()?
	_ = a
	return nil
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	// a, err := bar()  +  if err != nil { return err }
	assertContains(t, out, "a, err := bar()")
	assertContains(t, out, "if err != nil")
	assertContains(t, out, "return err")
	assertNotContains(t, out, "?")
}

func TestTranspileFile_QuestionOp_Expression(t *testing.T) {
	src := `package main

func foo() error {
	bar()?
	return nil
}

func bar() error { return nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	// if err := bar(); err != nil { return err }
	assertContains(t, out, "err := bar()")
	assertContains(t, out, "if err")
	assertContains(t, out, "return err")
	assertNotContains(t, out, "?")
}

func TestTranspileFile_QuestionOp_MultipleInSequence(t *testing.T) {
	src := `package main

func foo() error {
	a := step1()?
	b := step2()?
	_ = a
	_ = b
	return nil
}

func step1() (int, error) { return 0, nil }
func step2() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	if strings.Count(out, "if err != nil") != 2 {
		t.Errorf("expected 2 error checks\n\nOutput:\n%s", out)
	}
}

// ─── must ─────────────────────────────────────────────────────────────────────

func TestTranspileFile_Must_Assignment(t *testing.T) {
	src := `package main

func foo() {
	db := must open()
	_ = db
}

func open() (string, error) { return "", nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	// db, err := open()  +  if err != nil { panic(err) }
	assertContains(t, out, "db, err := open()")
	assertContains(t, out, "if err != nil")
	assertContains(t, out, "panic(err)")
	assertNotContains(t, out, "must ")
}

func TestTranspileFile_Must_Expression(t *testing.T) {
	src := `package main

func foo() {
	must validate()
}

func validate() error { return nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	// if err := validate(); err != nil { panic(err) }
	assertContains(t, out, "err := validate()")
	assertContains(t, out, "panic(err)")
	assertNotContains(t, out, "must ")
}

func TestTranspileFile_Must_DoesNotUseReturnErr(t *testing.T) {
	src := `package main

func foo() {
	must open()
}

func open() error { return nil }
`
	out := transpileOK(t, src)
	// must always panics, never returns
	assertContains(t, out, "panic(err)")
	assertNotContains(t, out, "return err")
}

// ─── throw ────────────────────────────────────────────────────────────────────

func TestTranspileFile_Throw_SimpleError(t *testing.T) {
	src := `package main

import "errors"

func foo() error {
	throw errors.New("oops")
}
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, `return errors.New("oops")`)
	assertNotContains(t, out, "throw ")
}

func TestTranspileFile_Throw_CallExpression(t *testing.T) {
	src := `package main

func foo() error {
	throw makeErr()
}

func makeErr() error { return nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "return makeErr()")
	assertNotContains(t, out, "throw ")
}

func TestTranspileFile_Throw_InIfBlock(t *testing.T) {
	src := `package main

import "errors"

func validate(s string) error {
	if s == "" {
		throw errors.New("empty string")
	}
	return nil
}
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, `return errors.New("empty string")`)
	assertNotContains(t, out, "throw ")
}

// ─── constructs in various contexts ───────────────────────────────────────────

func TestTranspileFile_TryCatch_InForLoop(t *testing.T) {
	src := `package main

func foo() {
	for i := 0; i < 10; i++ {
		try {
			@errcheck
			_, err := bar()
		} catch {
			continue
		}
	}
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "for i := 0")
	assertContains(t, out, "if err != nil")
	assertNotContains(t, out, "try {")
}

func TestTranspileFile_TryCatch_InIfBody(t *testing.T) {
	src := `package main

func foo(cond bool) error {
	if cond {
		try {
			@errcheck
			_, err := bar()
		} catch {
			return err
		}
	}
	return nil
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "if cond")
	assertContains(t, out, "if err != nil")
}

func TestTranspileFile_QuestionOp_InForLoop(t *testing.T) {
	src := `package main

func foo() error {
	for i := 0; i < 5; i++ {
		_ = step()?
	}
	return nil
}

func step() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "for i := 0")
	assertContains(t, out, "if err != nil")
}

// ─── multiple functions ───────────────────────────────────────────────────────

func TestTranspileFile_MultipleFunctions_EachTranspiled(t *testing.T) {
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

func baz() error {
	a := bar()?
	_ = a
	return nil
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	if strings.Count(out, "if err != nil") != 2 {
		t.Errorf("expected 2 error-check blocks (one per function)\n\nOutput:\n%s", out)
	}
	assertNotContains(t, out, "try {")
	assertNotContains(t, out, "?")
}

// ─── empty / edge cases ───────────────────────────────────────────────────────

func TestTranspileFile_EmptyFunction(t *testing.T) {
	src := `package main

func foo() {}
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "func foo()")
}

func TestTranspileFile_TryCatch_EmptyBody(t *testing.T) {
	src := `package main

func foo() {
	try {
	} catch {
	}
}
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertNotContains(t, out, "try {")
}

func TestTranspileFile_Annotations_NotInOutput(t *testing.T) {
	// Verify that no godsl-specific syntax leaks into the output
	src := `package main

import "errors"

func foo() error {
	try {
		@errcheck
		_, err := bar()
	} catch {
		return err
	}
	a := bar()?
	_ = a
	must bar()
	throw errors.New("x")
}

func bar() (int, error) { return 0, nil }
`
	out, _ := transpiler.TranspileFile(src)
	// Even if transpilation errors on throw-in-void-func, check what we get
	if out != "" {
		assertNotContains(t, out, "try {")
		assertNotContains(t, out, "} catch")
		assertNotContains(t, out, "} finally")
		assertNotContains(t, out, "@errcheck")
		assertNotContains(t, out, "throw ")
		assertNotContains(t, out, "must ")
	}
}

// ─── error cases ──────────────────────────────────────────────────────────────

func TestTranspileFile_InvalidSyntax_ReturnsError(t *testing.T) {
	src := `this is not valid godsl at all @@@@`
	_, err := transpiler.TranspileFile(src)
	if err == nil {
		t.Error("expected TranspileFile to return an error for invalid input, got nil")
	}
}

func TestTranspileFile_MissingPackage_ReturnsError(t *testing.T) {
	src := `func foo() {}`
	_, err := transpiler.TranspileFile(src)
	if err == nil {
		t.Error("expected TranspileFile to return an error for input without package clause")
	}
}

func TestTranspileFile_EmptyString_ReturnsError(t *testing.T) {
	_, err := transpiler.TranspileFile("")
	if err == nil {
		t.Error("expected TranspileFile to return an error for empty input")
	}
}

// ─── output does not contain godsl keywords ───────────────────────────────────

// ─── typed catch + finally (covers createTypeCheckIIFE) ──────────────────────

func TestTranspileFile_Finally_WithTypedCatch_NoReturn(t *testing.T) {
	src := `package main

import "fmt"

type MyError struct{}

func (e MyError) Error() string { return "my" }

func foo() {
	try {
		@errcheck
		_, err := bar()
	} catch(MyError) {
		fmt.Println("got MyError")
	} finally {
		fmt.Println("cleanup")
	}
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "MyError")
	assertContains(t, out, `fmt.Println("cleanup")`)
	assertNotContains(t, out, "finally")
}

func TestTranspileFile_Finally_WithTypedCatch_WithReturn(t *testing.T) {
	src := `package main

import "fmt"

type MyError struct{}

func (e MyError) Error() string { return "my" }

func foo() {
	try {
		@errcheck
		_, err := bar()
	} catch(MyError) {
		fmt.Println("got MyError")
		return
	} finally {
		fmt.Println("cleanup")
	}
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "MyError")
	assertContains(t, out, "_godslRet")
	assertContains(t, out, `fmt.Println("cleanup")`)
}

func TestTranspileFile_Finally_WithTypedCatch_WithVar_WithReturn(t *testing.T) {
	src := `package main

import "fmt"

type MyError struct{ Msg string }

func (e MyError) Error() string { return e.Msg }

func foo() {
	try {
		@errcheck
		_, err := bar()
	} catch(e MyError) {
		fmt.Println(e.Msg)
		return
	} finally {
		fmt.Println("cleanup")
	}
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "_godslRet")
	assertContains(t, out, `fmt.Println("cleanup")`)
}

func TestTranspileFile_Finally_MultiTypeCatch_WithReturn(t *testing.T) {
	src := `package main

import "fmt"

type ErrA struct{}
type ErrB struct{}

func (e ErrA) Error() string { return "a" }
func (e ErrB) Error() string { return "b" }

func foo() {
	try {
		@errcheck
		_, err := bar()
	} catch(ErrA | ErrB) {
		fmt.Println("caught multi")
		return
	} finally {
		fmt.Println("cleanup")
	}
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "_godslRet")
	assertContains(t, out, "ErrA")
	assertContains(t, out, "ErrB")
}

func TestTranspileFile_Finally_CatchAll_WithVar(t *testing.T) {
	src := `package main

import "fmt"

func foo() {
	try {
		@errcheck
		_, err := bar()
	} catch {
		fmt.Println(err)
	} finally {
		fmt.Println("done")
	}
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, `fmt.Println("done")`)
	assertNotContains(t, out, "finally")
}

func TestTranspileFile_Finally_NoCatch(t *testing.T) {
	src := `package main

import "fmt"

func foo() {
	try {
		@errcheck
		_, err := bar()
	} finally {
		fmt.Println("finally")
	}
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, `fmt.Println("finally")`)
}

// ─── transpileStmt coverage: switch, range, etc. ─────────────────────────────

func TestTranspileFile_SwitchStatement_Passthrough(t *testing.T) {
	src := `package main

func foo(x int) string {
	switch x {
	case 1:
		return "one"
	case 2:
		return "two"
	default:
		return "other"
	}
}
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "switch x")
	assertContains(t, out, `"one"`)
}

func TestTranspileFile_RangeLoop_Passthrough(t *testing.T) {
	src := `package main

func foo(items []string) {
	for i, v := range items {
		_ = i
		_ = v
	}
}
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "range items")
}

func TestTranspileFile_FuncWithNoBody_Passthrough(t *testing.T) {
	// Interface method declaration — FuncDecl with nil body
	src := `package main

type Writer interface {
	Write(p []byte) (n int, err error)
}

func foo() {}
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "Writer")
}

// ─── QuestionOp fallback (default case) ──────────────────────────────────────

func TestTranspileFile_QuestionOp_Assignment_MultiLhs(t *testing.T) {
	// a, b := f()? where f returns (T1, T2, error)
	src := `package main

func foo() error {
	a, b := multiReturn()?
	_ = a
	_ = b
	return nil
}

func multiReturn() (int, string, error) { return 0, "", nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "if err != nil")
}

// ─── must with re-assignment (= not :=) ──────────────────────────────────────

func TestTranspileFile_Must_ReassignmentForm(t *testing.T) {
	src := `package main

func foo() {
	var x string
	x = must produce()
	_ = x
}

func produce() (string, error) { return "", nil }
`
	// This form may or may not be supported, just ensure no panic
	out, err := transpiler.TranspileFile(src)
	_ = out
	_ = err
}

func TestTranspileFile_OutputIsValidGo_AllConstructs(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "try_catch",
			src: `package main
func foo() error {
	try { @errcheck; _, err := bar() } catch { return err }
	return nil
}
func bar() (int, error) { return 0, nil }`,
		},
		{
			name: "question_assign",
			src: `package main
func foo() error {
	a := bar()?; _ = a; return nil
}
func bar() (int, error) { return 0, nil }`,
		},
		{
			name: "must_assign",
			src: `package main
func foo() { db := must open(); _ = db }
func open() (string, error) { return "", nil }`,
		},
		{
			name: "throw",
			src: `package main
import "errors"
func foo() error { throw errors.New("e") }`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := transpiler.TranspileFile(tc.src)
			if err != nil {
				t.Fatalf("TranspileFile error: %v", err)
			}
			assertValidGo(t, out)
		})
	}
}

// ─── ternary operator ─────────────────────────────────────────────────────────

func TestTranspileFile_Ternary_BasicAssignment(t *testing.T) {
	src := `package main

func foo(x int) any {
	result := x > 0 ? "positive" : "non-positive"
	return result
}
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "if x > 0")
	assertContains(t, out, `"positive"`)
	assertContains(t, out, `"non-positive"`)
	assertNotContains(t, out, "?")
}

func TestTranspileFile_Ternary_ReturnExpr(t *testing.T) {
	src := `package main

func abs(x int) any {
	return x >= 0 ? x : -x
}
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "if x >= 0")
	assertContains(t, out, "return x")
	assertContains(t, out, "return -x")
	assertNotContains(t, out, "?")
}

func TestTranspileFile_Ternary_AsCallArgument(t *testing.T) {
	src := `package main

import "fmt"

func foo(flag bool) {
	fmt.Println(flag ? "yes" : "no")
}
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "fmt.Println")
	assertContains(t, out, `"yes"`)
	assertContains(t, out, `"no"`)
	assertNotContains(t, out, "?")
}

func TestTranspileFile_Ternary_Nested(t *testing.T) {
	src := `package main

func classify(x int) any {
	result := x > 0 ? "positive" : x < 0 ? "negative" : "zero"
	return result
}
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "if x > 0")
	assertContains(t, out, `"positive"`)
	assertContains(t, out, `"negative"`)
	assertContains(t, out, `"zero"`)
	assertNotContains(t, out, "?")
}

func TestTranspileFile_Ternary_WithBooleanCondition(t *testing.T) {
	src := `package main

func foo(a, b int) any {
	max := a > b ? a : b
	return max
}
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "if a > b")
	assertNotContains(t, out, "?")
}

func TestTranspileFile_Ternary_InsideTryCatch(t *testing.T) {
	src := `package main

import "fmt"

func foo(flag bool) error {
	try {
		@errcheck
		_, err := bar()
		fmt.Println(flag ? "ok" : "fail")
	} catch {
		return err
	}
	return nil
}

func bar() (int, error) { return 0, nil }
`
	out := transpileOK(t, src)
	assertValidGo(t, out)
	assertContains(t, out, "if err != nil")
	assertContains(t, out, `"ok"`)
	assertContains(t, out, `"fail"`)
	assertNotContains(t, out, "?")
}

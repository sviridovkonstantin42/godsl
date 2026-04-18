package scanner_test

import (
	"testing"

	"github.com/sviridovkonstantin42/godsl/internal/scanner"
	"github.com/sviridovkonstantin42/godsl/internal/token"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// scanAll scans src and returns all (tok, lit) pairs until EOF.
func scanAll(t *testing.T, src string) []struct {
	tok token.Token
	lit string
} {
	t.Helper()
	fset := token.NewFileSet()
	file := fset.AddFile("test.godsl", fset.Base(), len(src))

	var errs scanner.ErrorList
	var s scanner.Scanner
	s.Init(file, []byte(src), func(pos token.Position, msg string) {
		errs.Add(pos, msg)
	}, scanner.ScanComments)

	var result []struct {
		tok token.Token
		lit string
	}
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		// Skip semicolons injected by the scanner
		if tok == token.SEMICOLON && lit == "\n" {
			continue
		}
		result = append(result, struct {
			tok token.Token
			lit string
		}{tok, lit})
	}

	if len(errs) > 0 {
		t.Logf("scanner errors: %v", errs)
	}
	return result
}

// firstToken returns the first non-semicolon token from src.
func firstToken(t *testing.T, src string) (token.Token, string) {
	t.Helper()
	tokens := scanAll(t, src)
	if len(tokens) == 0 {
		t.Fatal("no tokens scanned")
	}
	return tokens[0].tok, tokens[0].lit
}

// ─── new keywords ─────────────────────────────────────────────────────────────

func TestScanner_TryKeyword(t *testing.T) {
	tok, lit := firstToken(t, "try")
	if tok != token.TRY {
		t.Errorf("expected TRY, got %s (lit=%q)", tok, lit)
	}
}

func TestScanner_CatchKeyword(t *testing.T) {
	tok, lit := firstToken(t, "catch")
	if tok != token.CATCH {
		t.Errorf("expected CATCH, got %s (lit=%q)", tok, lit)
	}
}

func TestScanner_FinallyKeyword(t *testing.T) {
	tok, lit := firstToken(t, "finally")
	if tok != token.FINALLY {
		t.Errorf("expected FINALLY, got %s (lit=%q)", tok, lit)
	}
}

func TestScanner_ThrowKeyword(t *testing.T) {
	tok, lit := firstToken(t, "throw")
	if tok != token.THROW {
		t.Errorf("expected THROW, got %s (lit=%q)", tok, lit)
	}
}

func TestScanner_MustKeyword(t *testing.T) {
	tok, lit := firstToken(t, "must")
	if tok != token.MUST {
		t.Errorf("expected MUST, got %s (lit=%q)", tok, lit)
	}
}

func TestScanner_QuestionMarkToken(t *testing.T) {
	tok, lit := firstToken(t, "?")
	if tok != token.QUESTION {
		t.Errorf("expected QUESTION, got %s (lit=%q)", tok, lit)
	}
}

func TestScanner_ErrCheckAnnotation(t *testing.T) {
	tok, _ := firstToken(t, "@errcheck")
	if tok != token.ERRCHECK {
		t.Errorf("expected ERRCHECK, got %s", tok)
	}
}

func TestScanner_ErrCheck_WithSpaceBefore(t *testing.T) {
	tokens := scanAll(t, "   @errcheck")
	if len(tokens) == 0 || tokens[0].tok != token.ERRCHECK {
		t.Errorf("expected ERRCHECK after whitespace, got %v", tokens)
	}
}

// ─── new keywords not confused with identifiers ───────────────────────────────

func TestScanner_TryLikeIdent_NotKeyword(t *testing.T) {
	// "trying" should be scanned as IDENT, not TRY
	tok, lit := firstToken(t, "trying")
	if tok != token.IDENT {
		t.Errorf("'trying' should be IDENT, got %s", tok)
	}
	if lit != "trying" {
		t.Errorf("expected lit='trying', got %q", lit)
	}
}

func TestScanner_CatchLikeIdent_NotKeyword(t *testing.T) {
	tok, lit := firstToken(t, "catches")
	if tok != token.IDENT {
		t.Errorf("'catches' should be IDENT, got %s", tok)
	}
	if lit != "catches" {
		t.Errorf("expected lit='catches', got %q", lit)
	}
}

func TestScanner_FinallyLikeIdent_NotKeyword(t *testing.T) {
	tok, _ := firstToken(t, "finalize")
	if tok != token.IDENT {
		t.Errorf("'finalize' should be IDENT, got %s", tok)
	}
}

// ─── new keywords alongside existing Go tokens ────────────────────────────────

func TestScanner_AllNewKeywords_Table(t *testing.T) {
	cases := []struct {
		src  string
		want token.Token
	}{
		{"try", token.TRY},
		{"catch", token.CATCH},
		{"finally", token.FINALLY},
		{"throw", token.THROW},
		{"must", token.MUST},
		{"?", token.QUESTION},
		{"@errcheck", token.ERRCHECK},
	}

	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			tok, _ := firstToken(t, tc.src)
			if tok != tc.want {
				t.Errorf("src=%q: expected %s, got %s", tc.src, tc.want, tok)
			}
		})
	}
}

func TestScanner_RegularGoKeywords_Unaffected(t *testing.T) {
	cases := []struct {
		src  string
		want token.Token
	}{
		{"if", token.IF},
		{"for", token.FOR},
		{"func", token.FUNC},
		{"return", token.RETURN},
		{"package", token.PACKAGE},
		{"import", token.IMPORT},
		{"var", token.VAR},
		{"const", token.CONST},
		{"type", token.TYPE},
		{"struct", token.STRUCT},
		{"interface", token.INTERFACE},
		{"go", token.GO},
		{"defer", token.DEFER},
		{"select", token.SELECT},
		{"switch", token.SWITCH},
		{"case", token.CASE},
		{"default", token.DEFAULT},
		{"chan", token.CHAN},
		{"map", token.MAP},
		{"range", token.RANGE},
		{"break", token.BREAK},
		{"continue", token.CONTINUE},
		{"goto", token.GOTO},
		{"fallthrough", token.FALLTHROUGH},
	}

	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			tok, _ := firstToken(t, tc.src)
			if tok != tc.want {
				t.Errorf("src=%q: expected %s, got %s", tc.src, tc.want, tok)
			}
		})
	}
}

func TestScanner_TryCatchSequence(t *testing.T) {
	src := `try { } catch { }`
	tokens := scanAll(t, src)

	wantToks := []token.Token{
		token.TRY,
		token.LBRACE,
		token.RBRACE,
		token.CATCH,
		token.LBRACE,
		token.RBRACE,
	}

	if len(tokens) < len(wantToks) {
		t.Fatalf("expected at least %d tokens, got %d: %v", len(wantToks), len(tokens), tokens)
	}
	for i, want := range wantToks {
		if tokens[i].tok != want {
			t.Errorf("token[%d]: expected %s, got %s", i, want, tokens[i].tok)
		}
	}
}

func TestScanner_TryCatchFinallySequence(t *testing.T) {
	src := `try { } catch { } finally { }`
	tokens := scanAll(t, src)

	found := make(map[token.Token]bool)
	for _, tok := range tokens {
		found[tok.tok] = true
	}

	for _, want := range []token.Token{token.TRY, token.CATCH, token.FINALLY} {
		if !found[want] {
			t.Errorf("expected token %s in output, not found", want)
		}
	}
}

func TestScanner_QuestionInExpression(t *testing.T) {
	src := `a := f()?`
	tokens := scanAll(t, src)

	// Should contain IDENT(:=) LPAREN RPAREN QUESTION
	var hasQuestion bool
	for _, tok := range tokens {
		if tok.tok == token.QUESTION {
			hasQuestion = true
		}
	}
	if !hasQuestion {
		t.Errorf("expected QUESTION token in %q, tokens: %v", src, tokens)
	}
}

func TestScanner_ErrCheckWithinTryBody(t *testing.T) {
	src := `@errcheck a, err := f()`
	tokens := scanAll(t, src)
	if len(tokens) == 0 || tokens[0].tok != token.ERRCHECK {
		t.Errorf("expected first token to be ERRCHECK\n\nTokens: %v", tokens)
	}
}

func TestScanner_CommentErrCheck_ScannedAsComment(t *testing.T) {
	src := `//@errcheck`
	tokens := scanAll(t, src)
	if len(tokens) == 0 || tokens[0].tok != token.COMMENT {
		t.Errorf("expected //@errcheck to be COMMENT token, got %v", tokens)
	}
	if tokens[0].lit != "//@errcheck" {
		t.Errorf("expected lit='//@errcheck', got %q", tokens[0].lit)
	}
}

func TestScanner_Integers_Unaffected(t *testing.T) {
	tok, lit := firstToken(t, "42")
	if tok != token.INT {
		t.Errorf("expected INT, got %s", tok)
	}
	if lit != "42" {
		t.Errorf("expected lit='42', got %q", lit)
	}
}

func TestScanner_StringLiteral_Unaffected(t *testing.T) {
	tok, lit := firstToken(t, `"hello"`)
	if tok != token.STRING {
		t.Errorf("expected STRING, got %s", tok)
	}
	if lit != `"hello"` {
		t.Errorf("expected lit='\"hello\"', got %q", lit)
	}
}

// ─── numeric literals ─────────────────────────────────────────────────────────

func TestScanner_FloatLiteral(t *testing.T) {
	tok, lit := firstToken(t, "3.14")
	if tok != token.FLOAT {
		t.Errorf("expected FLOAT, got %s (lit=%q)", tok, lit)
	}
}

func TestScanner_HexLiteral(t *testing.T) {
	tok, _ := firstToken(t, "0xFF")
	if tok != token.INT {
		t.Errorf("expected INT for hex literal, got %s", tok)
	}
}

func TestScanner_OctalLiteral(t *testing.T) {
	tok, _ := firstToken(t, "0o77")
	if tok != token.INT {
		t.Errorf("expected INT for octal literal, got %s", tok)
	}
}

func TestScanner_BinaryLiteral(t *testing.T) {
	tok, _ := firstToken(t, "0b1010")
	if tok != token.INT {
		t.Errorf("expected INT for binary literal, got %s", tok)
	}
}

func TestScanner_ImaginaryLiteral(t *testing.T) {
	tok, _ := firstToken(t, "1i")
	if tok != token.IMAG {
		t.Errorf("expected IMAG, got %s", tok)
	}
}

func TestScanner_FloatWithExponent(t *testing.T) {
	tok, _ := firstToken(t, "1e10")
	if tok != token.FLOAT {
		t.Errorf("expected FLOAT for 1e10, got %s", tok)
	}
}

// ─── string and char literals ─────────────────────────────────────────────────

func TestScanner_RawStringLiteral(t *testing.T) {
	tok, lit := firstToken(t, "`hello world`")
	if tok != token.STRING {
		t.Errorf("expected STRING for raw string, got %s", tok)
	}
	if lit != "`hello world`" {
		t.Errorf("expected backtick string, got %q", lit)
	}
}

func TestScanner_CharLiteral(t *testing.T) {
	tok, _ := firstToken(t, "'a'")
	if tok != token.CHAR {
		t.Errorf("expected CHAR, got %s", tok)
	}
}

func TestScanner_StringWithEscape(t *testing.T) {
	tok, lit := firstToken(t, `"hello\nworld"`)
	if tok != token.STRING {
		t.Errorf("expected STRING, got %s", tok)
	}
	if lit != `"hello\nworld"` {
		t.Errorf("unexpected lit: %q", lit)
	}
}

func TestScanner_StringWithUnicode(t *testing.T) {
	tok, _ := firstToken(t, `"привет"`)
	if tok != token.STRING {
		t.Errorf("expected STRING, got %s", tok)
	}
}

// ─── operators and delimiters ─────────────────────────────────────────────────

func TestScanner_Operators(t *testing.T) {
	cases := []struct {
		src  string
		want token.Token
	}{
		{"+", token.ADD},
		{"-", token.SUB},
		{"*", token.MUL},
		{"/", token.QUO},
		{"%", token.REM},
		{"&", token.AND},
		{"|", token.OR},
		{"^", token.XOR},
		{"<<", token.SHL},
		{">>", token.SHR},
		{"+=", token.ADD_ASSIGN},
		{"-=", token.SUB_ASSIGN},
		{"*=", token.MUL_ASSIGN},
		{"/=", token.QUO_ASSIGN},
		{":=", token.DEFINE},
		{"==", token.EQL},
		{"!=", token.NEQ},
		{"<", token.LSS},
		{">", token.GTR},
		{"<=", token.LEQ},
		{">=", token.GEQ},
		{"&&", token.LAND},
		{"||", token.LOR},
		{"++", token.INC},
		{"--", token.DEC},
		{"<-", token.ARROW},
		{"...", token.ELLIPSIS},
		{"(", token.LPAREN},
		{")", token.RPAREN},
		{"[", token.LBRACK},
		{"]", token.RBRACK},
		{"{", token.LBRACE},
		{"}", token.RBRACE},
		{",", token.COMMA},
		{".", token.PERIOD},
		{":", token.COLON},
	}

	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			tok, _ := firstToken(t, tc.src)
			if tok != tc.want {
				t.Errorf("src=%q: expected %s, got %s", tc.src, tc.want, tok)
			}
		})
	}
}

// ─── comments ─────────────────────────────────────────────────────────────────

func TestScanner_LineComment(t *testing.T) {
	src := "// this is a comment\nfoo"
	tokens := scanAll(t, src)
	// First token should be COMMENT (with ScanComments mode)
	if len(tokens) == 0 || tokens[0].tok != token.COMMENT {
		t.Errorf("expected COMMENT as first token, got %v", tokens)
	}
}

func TestScanner_BlockComment(t *testing.T) {
	src := "/* block comment */foo"
	tokens := scanAll(t, src)
	if len(tokens) == 0 || tokens[0].tok != token.COMMENT {
		t.Errorf("expected COMMENT as first token for block comment, got %v", tokens)
	}
}

func TestScanner_MultiLineBlockComment(t *testing.T) {
	src := "/* line1\nline2\nline3 */\nfoo"
	tokens := scanAll(t, src)
	hasFoo := false
	for _, tok := range tokens {
		if tok.tok == token.IDENT && tok.lit == "foo" {
			hasFoo = true
		}
	}
	if !hasFoo {
		t.Errorf("expected IDENT 'foo' after block comment, tokens: %v", tokens)
	}
}

// ─── ErrorList ────────────────────────────────────────────────────────────────

func TestErrorList_Add(t *testing.T) {
	var el scanner.ErrorList
	el.Add(token.Position{Filename: "test.go", Line: 1, Column: 1}, "test error")
	if len(el) != 1 {
		t.Errorf("expected 1 error, got %d", len(el))
	}
}

func TestErrorList_Error_Single(t *testing.T) {
	var el scanner.ErrorList
	el.Add(token.Position{Filename: "test.go", Line: 5, Column: 3}, "unexpected token")
	msg := el.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
	// Should contain file/line info
	if len(msg) < 5 {
		t.Errorf("error message too short: %q", msg)
	}
}

func TestErrorList_Error_Multiple(t *testing.T) {
	var el scanner.ErrorList
	el.Add(token.Position{Line: 1}, "first error")
	el.Add(token.Position{Line: 2}, "second error")
	msg := el.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
}

func TestErrorList_Err_Empty_ReturnsNil(t *testing.T) {
	var el scanner.ErrorList
	if el.Err() != nil {
		t.Error("expected nil for empty ErrorList")
	}
}

func TestErrorList_Err_NonEmpty_ReturnsError(t *testing.T) {
	var el scanner.ErrorList
	el.Add(token.Position{Line: 1}, "some error")
	if el.Err() == nil {
		t.Error("expected non-nil error for non-empty ErrorList")
	}
}

func TestErrorList_Sort(t *testing.T) {
	var el scanner.ErrorList
	el.Add(token.Position{Filename: "b.go", Line: 3}, "error at b:3")
	el.Add(token.Position{Filename: "a.go", Line: 1}, "error at a:1")
	el.Add(token.Position{Filename: "a.go", Line: 2}, "error at a:2")
	el.Sort()
	if el[0].Pos.Filename != "a.go" || el[0].Pos.Line != 1 {
		t.Errorf("expected first error at a.go:1, got %v", el[0].Pos)
	}
}

func TestErrorList_RemoveMultiples(t *testing.T) {
	var el scanner.ErrorList
	el.Add(token.Position{Filename: "a.go", Line: 1}, "first")
	el.Add(token.Position{Filename: "a.go", Line: 1}, "second on same line")
	el.Add(token.Position{Filename: "a.go", Line: 2}, "different line")
	el.RemoveMultiples()
	if len(el) != 2 {
		t.Errorf("expected 2 errors after RemoveMultiples (one per line), got %d", len(el))
	}
}

func TestErrorList_Reset(t *testing.T) {
	var el scanner.ErrorList
	el.Add(token.Position{Line: 1}, "error")
	el.Reset()
	if len(el) != 0 {
		t.Errorf("expected empty ErrorList after Reset, got %d", len(el))
	}
}

func TestErrorList_Len_Swap(t *testing.T) {
	var el scanner.ErrorList
	el.Add(token.Position{Line: 2}, "b")
	el.Add(token.Position{Line: 1}, "a")
	if el.Len() != 2 {
		t.Errorf("expected Len()=2, got %d", el.Len())
	}
	el.Swap(0, 1)
	if el[0].Pos.Line != 1 {
		t.Errorf("expected Line=1 after swap, got %d", el[0].Pos.Line)
	}
}

// ─── scanning with error handler ─────────────────────────────────────────────

func TestScanner_ErrorHandler_CalledOnBadToken(t *testing.T) {
	src := `@unknown`  // @unknown is not @errcheck → scanner error
	fset := token.NewFileSet()
	file := fset.AddFile("test.godsl", fset.Base(), len(src))

	var errCalled bool
	var s scanner.Scanner
	s.Init(file, []byte(src), func(pos token.Position, msg string) {
		errCalled = true
	}, 0)

	for {
		_, tok, _ := s.Scan()
		if tok == token.EOF {
			break
		}
	}

	if !errCalled {
		t.Error("expected error handler to be called for unknown @ annotation")
	}
}

func TestScanner_InitWithNilErrorHandler(t *testing.T) {
	src := `package main`
	fset := token.NewFileSet()
	file := fset.AddFile("test.godsl", fset.Base(), len(src))
	var s scanner.Scanner
	// nil error handler should not panic
	s.Init(file, []byte(src), nil, 0)
	_, tok, _ := s.Scan()
	if tok != token.PACKAGE {
		t.Errorf("expected PACKAGE, got %s", tok)
	}
}

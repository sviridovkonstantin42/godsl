package token_test

import (
	"testing"

	"github.com/sviridovkonstantin42/godsl/internal/token"
)

// ─── String() for new tokens ──────────────────────────────────────────────────

func TestToken_String_NewKeywords(t *testing.T) {
	cases := []struct {
		tok  token.Token
		want string
	}{
		{token.TRY, "try"},
		{token.CATCH, "catch"},
		{token.FINALLY, "finally"},
		{token.THROW, "throw"},
		{token.MUST, "must"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.tok.String(); got != tc.want {
				t.Errorf("Token(%d).String() = %q, want %q", tc.tok, got, tc.want)
			}
		})
	}
}

func TestToken_String_QUESTION(t *testing.T) {
	if got := token.QUESTION.String(); got != "?" {
		t.Errorf("QUESTION.String() = %q, want %q", got, "?")
	}
}

func TestToken_String_ERRCHECK(t *testing.T) {
	s := token.ERRCHECK.String()
	if s == "" {
		t.Error("ERRCHECK.String() returned empty string")
	}
}

// ─── Lookup() ─────────────────────────────────────────────────────────────────

func TestToken_Lookup_TryKeyword(t *testing.T) {
	if got := token.Lookup("try"); got != token.TRY {
		t.Errorf("Lookup(\"try\") = %s, want TRY", got)
	}
}

func TestToken_Lookup_CatchKeyword(t *testing.T) {
	if got := token.Lookup("catch"); got != token.CATCH {
		t.Errorf("Lookup(\"catch\") = %s, want CATCH", got)
	}
}

func TestToken_Lookup_FinallyKeyword(t *testing.T) {
	if got := token.Lookup("finally"); got != token.FINALLY {
		t.Errorf("Lookup(\"finally\") = %s, want FINALLY", got)
	}
}

func TestToken_Lookup_ThrowKeyword(t *testing.T) {
	if got := token.Lookup("throw"); got != token.THROW {
		t.Errorf("Lookup(\"throw\") = %s, want THROW", got)
	}
}

func TestToken_Lookup_MustKeyword(t *testing.T) {
	if got := token.Lookup("must"); got != token.MUST {
		t.Errorf("Lookup(\"must\") = %s, want MUST", got)
	}
}

func TestToken_Lookup_NewKeywords_Table(t *testing.T) {
	cases := []struct {
		name string
		want token.Token
	}{
		{"try", token.TRY},
		{"catch", token.CATCH},
		{"finally", token.FINALLY},
		{"throw", token.THROW},
		{"must", token.MUST},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := token.Lookup(tc.name)
			if got != tc.want {
				t.Errorf("Lookup(%q) = %s, want %s", tc.name, got, tc.want)
			}
		})
	}
}

func TestToken_Lookup_UnknownIdent_ReturnsIDENT(t *testing.T) {
	cases := []string{"foo", "bar", "myVar", "trying", "catches", "finalize", "thrower"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			got := token.Lookup(name)
			if got != token.IDENT {
				t.Errorf("Lookup(%q) = %s, want IDENT", name, got)
			}
		})
	}
}

func TestToken_Lookup_ExistingGoKeywords_Unaffected(t *testing.T) {
	cases := []struct {
		name string
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
		t.Run(tc.name, func(t *testing.T) {
			got := token.Lookup(tc.name)
			if got != tc.want {
				t.Errorf("Lookup(%q) = %s, want %s", tc.name, got, tc.want)
			}
		})
	}
}

// ─── IsKeyword() ──────────────────────────────────────────────────────────────

func TestToken_IsKeyword_NewKeywords(t *testing.T) {
	keywords := []token.Token{
		token.TRY,
		token.CATCH,
		token.FINALLY,
		token.THROW,
		token.MUST,
	}
	for _, tok := range keywords {
		t.Run(tok.String(), func(t *testing.T) {
			if !tok.IsKeyword() {
				t.Errorf("expected %s to be a keyword", tok)
			}
		})
	}
}

func TestToken_IsKeyword_NonKeywords(t *testing.T) {
	nonKeywords := []token.Token{
		token.IDENT,
		token.INT,
		token.FLOAT,
		token.STRING,
		token.ADD,
		token.SUB,
		token.LBRACE,
		token.RBRACE,
		token.QUESTION,
		token.ERRCHECK,
	}
	for _, tok := range nonKeywords {
		t.Run(tok.String(), func(t *testing.T) {
			if tok.IsKeyword() {
				t.Errorf("expected %s NOT to be a keyword", tok)
			}
		})
	}
}

func TestToken_IsKeyword_ByName(t *testing.T) {
	// token.IsKeyword(name) checks by name string
	cases := []struct {
		name string
		want bool
	}{
		{"try", true},
		{"catch", true},
		{"finally", true},
		{"throw", true},
		{"must", true},
		{"if", true},
		{"for", true},
		{"func", true},
		{"foo", false},
		{"bar", false},
		{"trying", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := token.IsKeyword(tc.name)
			if got != tc.want {
				t.Errorf("IsKeyword(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// ─── IsLiteral() and IsOperator() — new tokens should not be confused ─────────

func TestToken_IsLiteral_NewTokensAreNotLiterals(t *testing.T) {
	nonLiterals := []token.Token{
		token.TRY, token.CATCH, token.FINALLY, token.THROW, token.MUST,
		token.QUESTION, token.ERRCHECK,
	}
	for _, tok := range nonLiterals {
		t.Run(tok.String(), func(t *testing.T) {
			if tok.IsLiteral() {
				t.Errorf("expected %s NOT to be a literal", tok)
			}
		})
	}
}

func TestToken_IsOperator_NewKeywordsAreNotOperators(t *testing.T) {
	nonOps := []token.Token{
		token.TRY, token.CATCH, token.FINALLY, token.THROW, token.MUST,
	}
	for _, tok := range nonOps {
		t.Run(tok.String(), func(t *testing.T) {
			if tok.IsOperator() {
				t.Errorf("expected keyword %s NOT to be an operator", tok)
			}
		})
	}
}

// ─── ERRCHECK and QUESTION position in token list ─────────────────────────────

func TestToken_ERRCHECK_NotKeywordNotLiteralNotOperator(t *testing.T) {
	tok := token.ERRCHECK
	if tok.IsKeyword() {
		t.Error("ERRCHECK should not be classified as keyword")
	}
	if tok.IsLiteral() {
		t.Error("ERRCHECK should not be classified as literal")
	}
}

func TestToken_QUESTION_IsOperator(t *testing.T) {
	// QUESTION is in operator_beg..operator_end range
	if !token.QUESTION.IsOperator() {
		t.Error("expected QUESTION to be classified as an operator")
	}
}

// ─── NoPos sentinel ───────────────────────────────────────────────────────────

func TestToken_NoPos(t *testing.T) {
	if token.NoPos != 0 {
		t.Errorf("expected NoPos == 0, got %d", token.NoPos)
	}
	if token.NoPos.IsValid() {
		t.Error("NoPos.IsValid() should return false")
	}
}

// ─── Precedence ───────────────────────────────────────────────────────────────

func TestToken_Precedence_BinaryOps(t *testing.T) {
	cases := []struct {
		tok  token.Token
		want int
	}{
		{token.LOR, 1},
		{token.LAND, 2},
		{token.EQL, 3},
		{token.NEQ, 3},
		{token.LSS, 3},
		{token.GTR, 3},
		{token.LEQ, 3},
		{token.GEQ, 3},
		{token.ADD, 4},
		{token.SUB, 4},
		{token.OR, 4},
		{token.XOR, 4},
		{token.MUL, 5},
		{token.QUO, 5},
		{token.REM, 5},
		{token.SHL, 5},
		{token.SHR, 5},
		{token.AND, 5},
		{token.AND_NOT, 5},
	}
	for _, tc := range cases {
		t.Run(tc.tok.String(), func(t *testing.T) {
			got := tc.tok.Precedence()
			if got != tc.want {
				t.Errorf("%s.Precedence() = %d, want %d", tc.tok, got, tc.want)
			}
		})
	}
}

func TestToken_Precedence_NonBinaryOps_ReturnsLowest(t *testing.T) {
	// Non-binary-op tokens → LowestPrecedence
	for _, tok := range []token.Token{token.IDENT, token.LBRACE, token.TRY, token.CATCH} {
		if got := tok.Precedence(); got != token.LowestPrec {
			t.Errorf("%s.Precedence() = %d, want LowestPrec (%d)", tok, got, token.LowestPrec)
		}
	}
}

// ─── IsExported and IsIdentifier ──────────────────────────────────────────────

func TestToken_IsExported(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"Foo", true},
		{"FooBar", true},
		{"MyError", true},
		{"foo", false},
		{"fooBar", false},
		{"_foo", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := token.IsExported(tc.name)
			if got != tc.want {
				t.Errorf("IsExported(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestToken_IsIdentifier(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"foo", true},
		{"Foo", true},
		{"_foo", true},
		{"foo123", true},
		{"", false},
		{"123foo", false},
		// new keywords are NOT valid identifiers
		{"try", false},
		{"catch", false},
		{"finally", false},
		{"throw", false},
		{"must", false},
		// existing Go keywords are NOT valid identifiers
		{"if", false},
		{"for", false},
		{"func", false},
	}
	for _, tc := range cases {
		t.Run(tc.name+"="+func() string {
			if tc.want {
				return "ident"
			}
			return "not-ident"
		}(), func(t *testing.T) {
			got := token.IsIdentifier(tc.name)
			if got != tc.want {
				t.Errorf("IsIdentifier(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// ─── Position ─────────────────────────────────────────────────────────────────

func TestPosition_IsValid(t *testing.T) {
	valid := token.Position{Filename: "test.go", Line: 1, Column: 1}
	if !valid.IsValid() {
		t.Error("position with Line>0 should be valid")
	}
	invalid := token.Position{}
	if invalid.IsValid() {
		t.Error("zero position should not be valid")
	}
}

func TestPosition_String_ValidWithFile(t *testing.T) {
	pos := token.Position{Filename: "test.go", Line: 10, Column: 5}
	s := pos.String()
	if s == "" || s == "-" {
		t.Errorf("expected non-empty valid position string, got %q", s)
	}
}

func TestPosition_String_Invalid(t *testing.T) {
	pos := token.Position{}
	s := pos.String()
	if s != "-" {
		t.Errorf("invalid position string = %q, want \"-\"", s)
	}
}

func TestPosition_String_ValidNoFile(t *testing.T) {
	pos := token.Position{Line: 5, Column: 3}
	s := pos.String()
	if s == "" || s == "-" {
		t.Errorf("expected valid position string without filename, got %q", s)
	}
}

// ─── FileSet ──────────────────────────────────────────────────────────────────

func TestFileSet_AddFile_And_Position(t *testing.T) {
	fset := token.NewFileSet()
	src := "package main"
	file := fset.AddFile("test.go", -1, len(src))
	if file == nil {
		t.Fatal("expected non-nil file")
	}
	if file.Name() != "test.go" {
		t.Errorf("expected name 'test.go', got %q", file.Name())
	}
	if file.Size() != len(src) {
		t.Errorf("expected size=%d, got %d", len(src), file.Size())
	}
}

func TestFileSet_Base_Positive(t *testing.T) {
	fset := token.NewFileSet()
	base := fset.Base()
	if base < 1 {
		t.Errorf("expected Base >= 1, got %d", base)
	}
}

func TestFileSet_Position_ValidPos(t *testing.T) {
	fset := token.NewFileSet()
	src := "package main\nfunc foo() {}\n"
	file := fset.AddFile("test.go", -1, len(src))
	file.SetLinesForContent([]byte(src))

	pos := file.Pos(0)
	position := fset.Position(pos)
	if !position.IsValid() {
		t.Errorf("expected valid position, got %v", position)
	}
	if position.Filename != "test.go" {
		t.Errorf("expected filename 'test.go', got %q", position.Filename)
	}
}

func TestFile_LineCount(t *testing.T) {
	fset := token.NewFileSet()
	src := "line1\nline2\nline3\n"
	file := fset.AddFile("test.go", -1, len(src))
	file.SetLinesForContent([]byte(src))
	lc := file.LineCount()
	if lc != 3 {
		t.Errorf("expected 3 lines, got %d", lc)
	}
}

func TestFile_Pos_And_Offset(t *testing.T) {
	fset := token.NewFileSet()
	src := "abcdef"
	file := fset.AddFile("f.go", -1, len(src))
	// Pos at offset 3
	pos := file.Pos(3)
	if !pos.IsValid() {
		t.Error("expected valid pos from file.Pos(3)")
	}
	off := file.Offset(pos)
	if off != 3 {
		t.Errorf("expected offset=3, got %d", off)
	}
}

func TestFileSet_Iterate(t *testing.T) {
	fset := token.NewFileSet()
	fset.AddFile("a.go", -1, 10)
	fset.AddFile("b.go", -1, 20)

	var names []string
	fset.Iterate(func(f *token.File) bool {
		names = append(names, f.Name())
		return true
	})
	if len(names) != 2 {
		t.Errorf("expected 2 files, got %d: %v", len(names), names)
	}
}

func TestFileSet_File_ReturnsCorrectFile(t *testing.T) {
	fset := token.NewFileSet()
	src := "hello"
	file := fset.AddFile("myfile.go", -1, len(src))
	pos := file.Pos(0)
	got := fset.File(pos)
	if got == nil {
		t.Fatal("expected non-nil file for valid pos")
	}
	if got.Name() != "myfile.go" {
		t.Errorf("expected 'myfile.go', got %q", got.Name())
	}
}

// internal/transpiler/transpiler.go
package transpiler

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/sviridovkonstantin42/godsl/internal/format"

	"github.com/sviridovkonstantin42/godsl/internal/ast"

	"github.com/sviridovkonstantin42/godsl/internal/parser"

	"github.com/sviridovkonstantin42/godsl/internal/token"
)

type Transpiler struct {
	fset             *token.FileSet
	comments         []*ast.CommentGroup
	errcheckComments map[token.Pos]bool // Позиции комментариев @errcheck для удаления
	returnTypeHint   ast.Expr           // тип первого возвращаемого значения текущей функции
}

// NewTranspiler создает новый экземпляр транспилятора
func NewTranspiler() *Transpiler {
	return &Transpiler{
		fset:             token.NewFileSet(),
		errcheckComments: make(map[token.Pos]bool),
	}
}

func (t *Transpiler) Transpile(source string) (string, error) {
	file, err := parser.ParseFile(t.fset, "", source, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("parse error: %v", err)
	}

	t.comments = file.Comments

	newFile := t.transpileFile(file)

	newFile.Comments = t.filterComments(newFile.Comments)

	var buf bytes.Buffer
	err = format.Node(&buf, t.fset, newFile)
	if err != nil {
		return "", fmt.Errorf("format error: %v", err)
	}

	result := t.cleanupFormatting(buf.String())
	return result, nil
}

// filterComments удаляет комментарии @errcheck из результирующего кода
func (t *Transpiler) filterComments(commentGroups []*ast.CommentGroup) []*ast.CommentGroup {
	var filteredGroups []*ast.CommentGroup

	for _, group := range commentGroups {
		var filteredComments []*ast.Comment

		for _, comment := range group.List {
			// Проверяем, является ли это @errcheck комментарием
			if !t.isErrCheckComment(comment) {
				filteredComments = append(filteredComments, comment)
			}
		}

		// Добавляем группу только если в ней остались комментарии
		if len(filteredComments) > 0 {
			filteredGroups = append(filteredGroups, &ast.CommentGroup{
				List: filteredComments,
			})
		}
	}

	return filteredGroups
}

// isErrCheckComment проверяет, является ли комментарий @errcheck комментарием
func (t *Transpiler) isErrCheckComment(comment *ast.Comment) bool {
	return strings.Contains(comment.Text, "//@errcheck") ||
		strings.Contains(comment.Text, "// @errcheck") ||
		strings.TrimSpace(comment.Text) == "//@errcheck" ||
		strings.TrimSpace(comment.Text) == "// @errcheck"
}

// cleanupFormatting убирает лишние пустые строки и исправляет форматирование
func (t *Transpiler) cleanupFormatting(code string) string {
	lines := strings.Split(code, "\n")
	var result []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Пропускаем пустые строки внутри if блоков
		if trimmed == "" && i > 0 && i < len(lines)-1 {
			prevLine := strings.TrimSpace(lines[i-1])
			nextLine := strings.TrimSpace(lines[i+1])

			// Если предыдущая строка - открывающая скобка if, а следующая - код
			if strings.HasSuffix(prevLine, "{") && nextLine != "" && !strings.HasPrefix(nextLine, "}") {
				continue // Пропускаем пустую строку
			}
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// transpileFile транспилирует весь файл
func (t *Transpiler) transpileFile(file *ast.File) *ast.File {
	newFile := &ast.File{
		Doc:      file.Doc,
		Package:  file.Package,
		Name:     file.Name,
		Imports:  file.Imports,
		Comments: file.Comments, // Сначала копируем все комментарии
	}

	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			newDecl := t.transpileFuncDecl(funcDecl)
			newFile.Decls = append(newFile.Decls, newDecl)
		} else {
			newFile.Decls = append(newFile.Decls, decl)
		}
	}

	return newFile
}

// transpileFuncDecl транспилирует функцию
func (t *Transpiler) transpileFuncDecl(funcDecl *ast.FuncDecl) *ast.FuncDecl {
	if funcDecl.Body == nil {
		return funcDecl
	}

	prev := t.returnTypeHint
	t.returnTypeHint = extractFirstReturnType(funcDecl.Type)
	defer func() { t.returnTypeHint = prev }()

	newBody := &ast.BlockStmt{}
	newBody.List = t.transpileStmts(funcDecl.Body.List)

	return &ast.FuncDecl{
		Doc:  funcDecl.Doc,
		Recv: funcDecl.Recv,
		Name: funcDecl.Name,
		Type: funcDecl.Type,
		Body: newBody,
	}
}

// extractFirstReturnType возвращает тип первого не-error результата функции.
// Для func() (int, error) → *ast.Ident{Name:"int"}.
// Для func() error → nil (error не используем как подсказку для тернарного).
func extractFirstReturnType(funcType *ast.FuncType) ast.Expr {
	if funcType == nil || funcType.Results == nil || len(funcType.Results.List) == 0 {
		return nil
	}
	first := funcType.Results.List[0].Type
	if ident, ok := first.(*ast.Ident); ok && ident.Name == "error" {
		return nil
	}
	return first
}

// transpileStmts транспилирует список statements
func (t *Transpiler) transpileStmts(stmts []ast.Stmt) []ast.Stmt {
	var result []ast.Stmt

	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.TryStmt:
			transpiled := t.transpileTryStmt(s)
			result = append(result, transpiled...)
		case *ast.QuestionStmt:
			transpiled := t.transpileQuestionStmt(s)
			result = append(result, transpiled...)
		case *ast.MustStmt:
			transpiled := t.transpileMustStmt(s)
			result = append(result, transpiled...)
		default:
			newStmt := t.transpileStmt(stmt)
			result = append(result, newStmt)
		}
	}

	return result
}

// transpileStmt транспилирует отдельный statement
func (t *Transpiler) transpileStmt(stmt ast.Stmt) ast.Stmt {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		return &ast.BlockStmt{
			Lbrace: s.Lbrace,
			List:   t.transpileStmts(s.List),
			Rbrace: s.Rbrace,
		}
	case *ast.IfStmt:
		return &ast.IfStmt{
			If:   s.If,
			Init: s.Init,
			Cond: t.transpileExpr(s.Cond),
			Body: &ast.BlockStmt{
				List: t.transpileStmts(s.Body.List),
			},
			Else: s.Else,
		}
	case *ast.ForStmt:
		return &ast.ForStmt{
			For:  s.For,
			Init: s.Init,
			Cond: t.transpileExpr(s.Cond),
			Post: s.Post,
			Body: &ast.BlockStmt{
				List: t.transpileStmts(s.Body.List),
			},
		}
	case *ast.AssignStmt:
		newLhs := make([]ast.Expr, len(s.Lhs))
		for i, e := range s.Lhs {
			newLhs[i] = t.transpileExpr(e)
		}
		newRhs := make([]ast.Expr, len(s.Rhs))
		for i, e := range s.Rhs {
			newRhs[i] = t.transpileExpr(e)
		}
		return &ast.AssignStmt{Lhs: newLhs, TokPos: s.TokPos, Tok: s.Tok, Rhs: newRhs}
	case *ast.ExprStmt:
		return &ast.ExprStmt{X: t.transpileExpr(s.X)}
	case *ast.ReturnStmt:
		newResults := make([]ast.Expr, len(s.Results))
		for i, e := range s.Results {
			newResults[i] = t.transpileExpr(e)
		}
		return &ast.ReturnStmt{Return: s.Return, Results: newResults}
	case *ast.SendStmt:
		return &ast.SendStmt{Chan: t.transpileExpr(s.Chan), Arrow: s.Arrow, Value: t.transpileExpr(s.Value)}
	case *ast.ThrowStmt:
		return t.transpileThrowStmt(s)
	default:
		return stmt
	}
}

// transpileExpr рекурсивно обходит выражение, заменяя TernaryExpr на IIFE.
func (t *Transpiler) transpileExpr(expr ast.Expr) ast.Expr {
	if expr == nil {
		return nil
	}
	switch x := expr.(type) {
	case *ast.TernaryExpr:
		return t.transpileTernaryExpr(x)
	case *ast.BinaryExpr:
		newX := t.transpileExpr(x.X)
		newY := t.transpileExpr(x.Y)
		if newX == x.X && newY == x.Y {
			return x
		}
		return &ast.BinaryExpr{X: newX, OpPos: x.OpPos, Op: x.Op, Y: newY}
	case *ast.UnaryExpr:
		newX := t.transpileExpr(x.X)
		if newX == x.X {
			return x
		}
		return &ast.UnaryExpr{OpPos: x.OpPos, Op: x.Op, X: newX}
	case *ast.StarExpr:
		newX := t.transpileExpr(x.X)
		if newX == x.X {
			return x
		}
		return &ast.StarExpr{Star: x.Star, X: newX}
	case *ast.ParenExpr:
		newX := t.transpileExpr(x.X)
		if newX == x.X {
			return x
		}
		return &ast.ParenExpr{Lparen: x.Lparen, X: newX, Rparen: x.Rparen}
	case *ast.CallExpr:
		changed := false
		newFun := t.transpileExpr(x.Fun)
		if newFun != x.Fun {
			changed = true
		}
		newArgs := make([]ast.Expr, len(x.Args))
		for i, arg := range x.Args {
			newArgs[i] = t.transpileExpr(arg)
			if newArgs[i] != arg {
				changed = true
			}
		}
		if !changed {
			return x
		}
		return &ast.CallExpr{Fun: newFun, Lparen: x.Lparen, Args: newArgs, Ellipsis: x.Ellipsis, Rparen: x.Rparen}
	case *ast.IndexExpr:
		newX := t.transpileExpr(x.X)
		newIdx := t.transpileExpr(x.Index)
		if newX == x.X && newIdx == x.Index {
			return x
		}
		return &ast.IndexExpr{X: newX, Lbrack: x.Lbrack, Index: newIdx, Rbrack: x.Rbrack}
	case *ast.SliceExpr:
		newX := t.transpileExpr(x.X)
		newLow := t.transpileExpr(x.Low)
		newHigh := t.transpileExpr(x.High)
		newMax := t.transpileExpr(x.Max)
		if newX == x.X && newLow == x.Low && newHigh == x.High && newMax == x.Max {
			return x
		}
		return &ast.SliceExpr{X: newX, Lbrack: x.Lbrack, Low: newLow, High: newHigh, Max: newMax, Slice3: x.Slice3, Rbrack: x.Rbrack}
	case *ast.KeyValueExpr:
		newKey := t.transpileExpr(x.Key)
		newVal := t.transpileExpr(x.Value)
		if newKey == x.Key && newVal == x.Value {
			return x
		}
		return &ast.KeyValueExpr{Key: newKey, Colon: x.Colon, Value: newVal}
	case *ast.CompositeLit:
		changed := false
		newElts := make([]ast.Expr, len(x.Elts))
		for i, elt := range x.Elts {
			newElts[i] = t.transpileExpr(elt)
			if newElts[i] != elt {
				changed = true
			}
		}
		if !changed {
			return x
		}
		return &ast.CompositeLit{Type: x.Type, Lbrace: x.Lbrace, Elts: newElts, Rbrace: x.Rbrace}
	case *ast.SelectorExpr:
		newX := t.transpileExpr(x.X)
		if newX == x.X {
			return x
		}
		return &ast.SelectorExpr{X: newX, Sel: x.Sel}
	default:
		return expr
	}
}

// transpileTernaryExpr преобразует cond ? then : else в:
//
//	func() T { if cond { return then }; return else }()
//
// Тип T выводится из ветвей (литералы) или из контекста возвращаемого типа
// текущей функции. Если вывести тип невозможно, используется any.
func (t *Transpiler) transpileTernaryExpr(x *ast.TernaryExpr) ast.Expr {
	cond := t.transpileExpr(x.Cond)
	then := t.transpileExpr(x.Then)
	els := t.transpileExpr(x.Else)

	retType := t.inferTernaryReturnType(x.Then, x.Else)
	if retType == nil {
		retType = &ast.Ident{NamePos: token.NoPos, Name: "any"}
	}

	return &ast.CallExpr{
		Fun: &ast.FuncLit{
			Type: &ast.FuncType{
				Func:   token.NoPos,
				Params: &ast.FieldList{},
				Results: &ast.FieldList{
					List: []*ast.Field{{
						Type: retType,
					}},
				},
			},
			Body: &ast.BlockStmt{
				Lbrace: token.NoPos,
				List: []ast.Stmt{
					&ast.IfStmt{
						If:   token.NoPos,
						Cond: cond,
						Body: &ast.BlockStmt{
							Lbrace: token.NoPos,
							List: []ast.Stmt{
								&ast.ReturnStmt{
									Return:  token.NoPos,
									Results: []ast.Expr{then},
								},
							},
							Rbrace: token.NoPos,
						},
					},
					&ast.ReturnStmt{
						Return:  token.NoPos,
						Results: []ast.Expr{els},
					},
				},
				Rbrace: token.NoPos,
			},
		},
		Lparen: token.NoPos,
		Rparen: token.NoPos,
	}
}

// inferTernaryReturnType выводит тип возвращаемого значения тернарного оператора.
//
// Порядок приоритетов:
//  1. Оба литерала одного вида (int/float/string/rune/bool) → этот тип.
//  2. Один из литералов известного типа → его тип.
//  3. Контекст: возвращаемый тип текущей функции (t.returnTypeHint).
//  4. Иначе → nil (будет подставлено any).
func (t *Transpiler) inferTernaryReturnType(then, els ast.Expr) ast.Expr {
	thenType := inferLiteralType(then)
	elsType := inferLiteralType(els)

	if thenType != nil && elsType != nil {
		// Оба типа выведены из литералов: используем тип ветки then
		// (если ветки конфликтуют, компилятор Go всё равно выдаст ошибку).
		return thenType
	}
	if thenType != nil {
		return thenType
	}
	if elsType != nil {
		return elsType
	}
	// Если ветки не содержат литералов — используем подсказку из контекста функции.
	return t.returnTypeHint
}

// inferLiteralType пытается вывести Go-тип выражения по его синтаксису.
// Работает без таблицы символов: распознаёт литералы, булевы константы и
// унарный минус/NOT применённый к ним.
// Возвращает nil, если тип определить невозможно.
func inferLiteralType(expr ast.Expr) ast.Expr {
	switch e := expr.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.INT:
			return &ast.Ident{NamePos: token.NoPos, Name: "int"}
		case token.FLOAT:
			return &ast.Ident{NamePos: token.NoPos, Name: "float64"}
		case token.STRING:
			return &ast.Ident{NamePos: token.NoPos, Name: "string"}
		case token.CHAR:
			return &ast.Ident{NamePos: token.NoPos, Name: "rune"}
		}
	case *ast.Ident:
		if e.Name == "true" || e.Name == "false" {
			return &ast.Ident{NamePos: token.NoPos, Name: "bool"}
		}
	case *ast.UnaryExpr:
		// -1, -3.14, ^0xff, !flag
		if e.Op == token.SUB || e.Op == token.XOR || e.Op == token.NOT {
			return inferLiteralType(e.X)
		}
	case *ast.ParenExpr:
		return inferLiteralType(e.X)
	}
	return nil
}

// transpileThrowStmt транспилирует throw <expr> → return <expr>
func (t *Transpiler) transpileThrowStmt(s *ast.ThrowStmt) ast.Stmt {
	return &ast.ReturnStmt{
		Return:  token.NoPos,
		Results: []ast.Expr{s.X},
	}
}

// transpileQuestionStmt транспилирует stmt? → stmt + if err != nil { return err }
// Для AssignStmt (a := f()?): добавляет err в левую часть
// Для ExprStmt (f()?): генерирует if err := f(); err != nil { return err }
func (t *Transpiler) transpileQuestionStmt(s *ast.QuestionStmt) []ast.Stmt {
	switch inner := s.Stmt.(type) {
	case *ast.AssignStmt:
		// a := readFile()? → a, err := readFile(); if err != nil { return err }
		newAssign := &ast.AssignStmt{
			Lhs: append(inner.Lhs, &ast.Ident{NamePos: token.NoPos, Name: "err"}),
			TokPos: inner.TokPos,
			Tok:    inner.Tok,
			Rhs:    inner.Rhs,
		}
		errCheck := t.createErrorCheck(nil)
		return []ast.Stmt{newAssign, errCheck}

	case *ast.ExprStmt:
		// f()? → if err := f(); err != nil { return err }
		ifStmt := &ast.IfStmt{
			If: token.NoPos,
			Init: &ast.AssignStmt{
				Lhs:    []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "err"}},
				TokPos: token.NoPos,
				Tok:    token.DEFINE,
				Rhs:    []ast.Expr{inner.X},
			},
			Cond: &ast.BinaryExpr{
				X:     &ast.Ident{NamePos: token.NoPos, Name: "err"},
				OpPos: token.NoPos,
				Op:    token.NEQ,
				Y:     &ast.Ident{NamePos: token.NoPos, Name: "nil"},
			},
			Body: &ast.BlockStmt{
				Lbrace: token.NoPos,
				List: []ast.Stmt{
					&ast.ReturnStmt{
						Return:  token.NoPos,
						Results: []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "err"}},
					},
				},
				Rbrace: token.NoPos,
			},
		}
		return []ast.Stmt{ifStmt}

	default:
		// Фолбэк: просто возвращаем оригинальный statement
		return []ast.Stmt{s.Stmt}
	}
}

// transpileMustStmt транспилирует MustStmt:
//
//	db := must sql.Open(...) → db, err := sql.Open(...); if err != nil { panic(err) }
//	must f()                 → if err := f(); err != nil { panic(err) }
func (t *Transpiler) transpileMustStmt(s *ast.MustStmt) []ast.Stmt {
	switch inner := s.Stmt.(type) {
	case *ast.AssignStmt:
		// db := must sql.Open(...) → db, err := sql.Open(...); if err != nil { panic(err) }
		newAssign := &ast.AssignStmt{
			Lhs:    append(inner.Lhs, &ast.Ident{NamePos: token.NoPos, Name: "err"}),
			TokPos: inner.TokPos,
			Tok:    inner.Tok,
			Rhs:    inner.Rhs,
		}
		panicCheck := &ast.IfStmt{
			If: token.NoPos,
			Cond: &ast.BinaryExpr{
				X:     &ast.Ident{NamePos: token.NoPos, Name: "err"},
				OpPos: token.NoPos,
				Op:    token.NEQ,
				Y:     &ast.Ident{NamePos: token.NoPos, Name: "nil"},
			},
			Body: &ast.BlockStmt{
				Lbrace: token.NoPos,
				List:   []ast.Stmt{createPanicErr()},
				Rbrace: token.NoPos,
			},
		}
		return []ast.Stmt{newAssign, panicCheck}

	case *ast.ExprStmt:
		// must f() → if err := f(); err != nil { panic(err) }
		ifStmt := &ast.IfStmt{
			If: token.NoPos,
			Init: &ast.AssignStmt{
				Lhs:    []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "err"}},
				TokPos: token.NoPos,
				Tok:    token.DEFINE,
				Rhs:    []ast.Expr{inner.X},
			},
			Cond: &ast.BinaryExpr{
				X:     &ast.Ident{NamePos: token.NoPos, Name: "err"},
				OpPos: token.NoPos,
				Op:    token.NEQ,
				Y:     &ast.Ident{NamePos: token.NoPos, Name: "nil"},
			},
			Body: &ast.BlockStmt{
				Lbrace: token.NoPos,
				List:   []ast.Stmt{createPanicErr()},
				Rbrace: token.NoPos,
			},
		}
		return []ast.Stmt{ifStmt}

	default:
		return []ast.Stmt{s.Stmt}
	}
}

// createPanicErr создаёт вызов panic(err).
func createPanicErr() ast.Stmt {
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun:  &ast.Ident{NamePos: token.NoPos, Name: "panic"},
			Args: []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "err"}},
		},
	}
}

// transpileTryStmt транспилирует TryStmt в обычные Go конструкции
func (t *Transpiler) transpileTryStmt(tryStmt *ast.TryStmt) []ast.Stmt {
	if tryStmt.Finally == nil {
		// Без finally — простая транспиляция как раньше
		return t.transpileTryCatchOnly(tryStmt)
	}
	// С finally — используем IIFE чтобы finally выполнялся
	// сразу после try-catch, а не в конце всей функции
	return t.transpileTryCatchFinally(tryStmt)
}

// transpileTryCatchOnly транспилирует try-catch без finally (оригинальная логика)
func (t *Transpiler) transpileTryCatchOnly(tryStmt *ast.TryStmt) []ast.Stmt {
	var result []ast.Stmt
	for _, stmt := range tryStmt.Body.List {
		if ec, ok := stmt.(*ast.ErrCheckStmt); ok {
			// @errcheck — синтаксическая аннотация
			result = append(result, t.transpileStmt(ec.Stmt))
			result = append(result, t.createErrorCheck(tryStmt.Catches))
		} else {
			result = append(result, t.transpileStmt(stmt))
			if t.hasErrCheckComment(stmt) {
				// //@errcheck — старый комментарий-синтаксис
				result = append(result, t.createErrorCheck(tryStmt.Catches))
			}
		}
	}
	return result
}

// transpileTryCatchFinally транспилирует try-catch-finally через IIFE:
//
//	_ret := func() bool { <try-catch> }()
//	<finally>
//	if _ret { return }
//
// Это гарантирует что finally выполнится сразу после try-catch блока,
// до любого кода после него, и при этом return в catch корректно
// распространяется на внешнюю функцию.
func (t *Transpiler) transpileTryCatchFinally(tryStmt *ast.TryStmt) []ast.Stmt {
	catchesHaveReturn := t.catchesHaveReturn(tryStmt.Catches)

	// Тело IIFE: try-логика + return false в конце
	var iifeBody []ast.Stmt
	for _, stmt := range tryStmt.Body.List {
		if ec, ok := stmt.(*ast.ErrCheckStmt); ok {
			iifeBody = append(iifeBody, t.transpileStmt(ec.Stmt))
			iifeBody = append(iifeBody, t.createErrorCheckIIFE(tryStmt.Catches, catchesHaveReturn))
		} else {
			iifeBody = append(iifeBody, t.transpileStmt(stmt))
			if t.hasErrCheckComment(stmt) {
				iifeBody = append(iifeBody, t.createErrorCheckIIFE(tryStmt.Catches, catchesHaveReturn))
			}
		}
	}
	iifeBody = append(iifeBody, &ast.ReturnStmt{
		Return:  token.NoPos,
		Results: []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "false"}},
	})

	// _ret := func() bool { ... }()
	var result []ast.Stmt
	if catchesHaveReturn {
		result = append(result, &ast.AssignStmt{
			Lhs:    []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "_godslRet"}},
			TokPos: token.NoPos,
			Tok:    token.DEFINE,
			Rhs: []ast.Expr{
				t.makeBoolIIFE(iifeBody, tryStmt.Try),
			},
		})
	} else {
		// Нет return в catch — вызываем IIFE без захвата результата
		result = append(result, &ast.ExprStmt{
			X: t.makeBoolIIFE(iifeBody, tryStmt.Try),
		})
	}

	// Добавляем finally-тело сразу после IIFE
	result = append(result, tryStmt.Finally.List...)

	// Если catch мог вернуть true — распространяем return
	if catchesHaveReturn {
		result = append(result, &ast.IfStmt{
			If: token.NoPos,
			Cond: &ast.Ident{NamePos: token.NoPos, Name: "_godslRet"},
			Body: &ast.BlockStmt{
				Lbrace: token.NoPos,
				List:   []ast.Stmt{&ast.ReturnStmt{Return: token.NoPos}},
				Rbrace: token.NoPos,
			},
		})
	}

	return result
}

// makeBoolIIFE создаёт func() bool { ... }() с заданным телом
func (t *Transpiler) makeBoolIIFE(body []ast.Stmt, pos token.Pos) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.FuncLit{
			Type: &ast.FuncType{
				Func:   pos,
				Params: &ast.FieldList{Opening: pos, Closing: pos},
				Results: &ast.FieldList{
					List: []*ast.Field{
						{Type: &ast.Ident{NamePos: pos, Name: "bool"}},
					},
				},
			},
			Body: &ast.BlockStmt{
				Lbrace: pos,
				List:   body,
				Rbrace: pos,
			},
		},
		Lparen: pos,
		Rparen: pos,
	}
}

// catchesHaveReturn проверяет, есть ли в catch-блоках оператор return
func (t *Transpiler) catchesHaveReturn(catches []*ast.CatchStmt) bool {
	for _, c := range catches {
		for _, stmt := range c.Body.List {
			if _, ok := stmt.(*ast.ReturnStmt); ok {
				return true
			}
		}
	}
	return false
}

// createErrorCheckIIFE — как createErrorCheck, но внутри IIFE:
// return в catch заменяется на return true, иначе return false не добавляется
func (t *Transpiler) createErrorCheckIIFE(catches []*ast.CatchStmt, catchesHaveReturn bool) ast.Stmt {
	var catchBody []ast.Stmt

	if len(catches) == 0 {
		catchBody = append(catchBody, &ast.ReturnStmt{
			Return:  token.NoPos,
			Results: []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "true"}},
		})
	} else {
		for i, catchStmt := range catches {
			if len(catchStmt.ErrorTypes) > 0 {
				typeCheck := t.createTypeCheckIIFE(catchStmt, i == len(catches)-1, catchesHaveReturn)
				catchBody = append(catchBody, typeCheck)
			} else {
				// catch-all: заменяем ReturnStmt → return true
				body := t.replaceCatchReturns(catchStmt.Body.List, catchesHaveReturn)
				if catchStmt.ErrorVar != nil {
					assignment := &ast.AssignStmt{
						Lhs:    []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: catchStmt.ErrorVar.Name}},
						TokPos: token.NoPos,
						Tok:    token.ASSIGN,
						Rhs:    []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "err"}},
					}
					catchBody = append(catchBody, assignment)
				}
				catchBody = append(catchBody, body...)
				break
			}
		}
	}

	return &ast.IfStmt{
		If: token.NoPos,
		Cond: &ast.BinaryExpr{
			X:     &ast.Ident{NamePos: token.NoPos, Name: "err"},
			OpPos: token.NoPos,
			Op:    token.NEQ,
			Y:     &ast.Ident{NamePos: token.NoPos, Name: "nil"},
		},
		Body: &ast.BlockStmt{
			Lbrace: token.NoPos,
			List:   catchBody,
			Rbrace: token.NoPos,
		},
	}
}

// createTypeCheckIIFE — как createTypeCheck, но в IIFE-контексте
func (t *Transpiler) createTypeCheckIIFE(catchStmt *ast.CatchStmt, isLast bool, catchesHaveReturn bool) ast.Stmt {
	var condition ast.Expr
	var init ast.Stmt
	var extraBodyPrefix []ast.Stmt

	if len(catchStmt.ErrorTypes) > 1 {
		// Multi-type: условие через IIFE func() bool { ... }()
		condition = t.makeMultiTypeCheck(catchStmt.ErrorTypes)
		if catchStmt.ErrorVar != nil {
			// e := err  (тип — error, конкретный тип не известен)
			extraBodyPrefix = []ast.Stmt{&ast.AssignStmt{
				Lhs:    []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: catchStmt.ErrorVar.Name}},
				TokPos: token.NoPos,
				Tok:    token.DEFINE,
				Rhs:    []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "err"}},
			}}
		}
	} else {
		// Single type
		typ := catchStmt.ErrorTypes[0]
		if catchStmt.ErrorVar != nil {
			init = &ast.AssignStmt{
				Lhs:    []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: catchStmt.ErrorVar.Name}, &ast.Ident{NamePos: token.NoPos, Name: "ok"}},
				TokPos: token.NoPos,
				Tok:    token.DEFINE,
				Rhs:    []ast.Expr{&ast.TypeAssertExpr{X: &ast.Ident{NamePos: token.NoPos, Name: "err"}, Type: typ}},
			}
		} else {
			init = &ast.AssignStmt{
				Lhs:    []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "_"}, &ast.Ident{NamePos: token.NoPos, Name: "ok"}},
				TokPos: token.NoPos,
				Tok:    token.DEFINE,
				Rhs:    []ast.Expr{&ast.TypeAssertExpr{X: &ast.Ident{NamePos: token.NoPos, Name: "err"}, Type: typ}},
			}
		}
		condition = &ast.Ident{NamePos: token.NoPos, Name: "ok"}
	}

	body := append(extraBodyPrefix, t.replaceCatchReturns(catchStmt.Body.List, catchesHaveReturn)...)

	ifStmt := &ast.IfStmt{
		If:   token.NoPos,
		Init: init,
		Cond: condition,
		Body: &ast.BlockStmt{Lbrace: token.NoPos, List: body, Rbrace: token.NoPos},
	}

	if !isLast {
		ifStmt.Else = &ast.BlockStmt{
			List: []ast.Stmt{&ast.ReturnStmt{
				Return:  token.NoPos,
				Results: []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "true"}},
			}},
		}
	}

	return ifStmt
}

// replaceCatchReturns заменяет ReturnStmt в теле catch на return true (для IIFE)
func (t *Transpiler) replaceCatchReturns(stmts []ast.Stmt, catchesHaveReturn bool) []ast.Stmt {
	if !catchesHaveReturn {
		return stmts
	}
	var result []ast.Stmt
	for _, stmt := range stmts {
		if _, ok := stmt.(*ast.ReturnStmt); ok {
			result = append(result, &ast.ReturnStmt{
				Return:  token.NoPos,
				Results: []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "true"}},
			})
		} else {
			result = append(result, stmt)
		}
	}
	return result
}

// hasErrCheckComment проверяет, есть ли у statement комментарий //@errcheck
func (t *Transpiler) hasErrCheckComment(stmt ast.Stmt) bool {
	stmtPos := stmt.Pos()
	stmtEnd := stmt.End()

	// Ищем комментарии в той же строке или на строке выше
	for _, commentGroup := range t.comments {
		for _, comment := range commentGroup.List {
			// Проверяем, что комментарий находится рядом со statement
			if t.isCommentRelatedToStmt(comment, stmtPos, stmtEnd) {
				if t.isErrCheckComment(comment) {
					// Помечаем этот комментарий для удаления
					t.errcheckComments[comment.Pos()] = true
					return true
				}
			}
		}
	}
	return false
}

// isCommentRelatedToStmt проверяет, относится ли комментарий к данному statement
func (t *Transpiler) isCommentRelatedToStmt(comment *ast.Comment, stmtPos, stmtEnd token.Pos) bool {
	commentPos := comment.Pos()

	// Получаем позиции в исходном коде
	stmtPosition := t.fset.Position(stmtPos)
	commentPosition := t.fset.Position(commentPos)

	// Комментарий должен быть в той же строке или на строке выше
	lineDiff := stmtPosition.Line - commentPosition.Line

	// Комментарий в той же строке (справа от кода)
	if lineDiff == 0 && commentPos > stmtPos {
		return true
	}

	// Комментарий на строке выше
	if lineDiff == 1 {
		return true
	}

	return false
}

// Оставляем старые функции для совместимости, но они больше не используются
// isErrorProducingStmt проверяет, может ли statement вернуть ошибку (не используется)
func (t *Transpiler) isErrorProducingStmt(stmt ast.Stmt) bool {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		// Проверяем присваивания вида: var, err := someFunc()
		if len(s.Lhs) >= 2 {
			if ident, ok := s.Lhs[len(s.Lhs)-1].(*ast.Ident); ok && ident.Name == "err" {
				return true
			}
		}
		// Проверяем присваивания вида: err = someFunc()
		for _, lhs := range s.Lhs {
			if ident, ok := lhs.(*ast.Ident); ok && ident.Name == "err" {
				return true
			}
		}
	}
	return false
}

// createErrorCheck создает блок проверки ошибки с catch обработчиками
func (t *Transpiler) createErrorCheck(catches []*ast.CatchStmt) ast.Stmt {
	var catchBody []ast.Stmt

	if len(catches) == 0 {
		// Если нет catch блоков, просто return err
		catchBody = append(catchBody, &ast.ReturnStmt{
			Return: token.NoPos,
			Results: []ast.Expr{
				&ast.Ident{
					NamePos: token.NoPos,
					Name:    "err",
				},
			},
		})
	} else {
		// Обрабатываем catch блоки
		for i, catchStmt := range catches {
			if len(catchStmt.ErrorTypes) > 0 {
				// Специфичный catch для определенного типа ошибки
				typeCheck := t.createTypeCheck(catchStmt, i == len(catches)-1)
				catchBody = append(catchBody, typeCheck)
			} else {
				// Catch-all - добавляем тело напрямую
				if catchStmt.ErrorVar != nil {
					// Присваиваем ошибку переменной из catch
					assignment := &ast.AssignStmt{
						Lhs: []ast.Expr{&ast.Ident{
							NamePos: token.NoPos,
							Name:    catchStmt.ErrorVar.Name,
						}},
						TokPos: token.NoPos,
						Tok:    token.ASSIGN,
						Rhs: []ast.Expr{&ast.Ident{
							NamePos: token.NoPos,
							Name:    "err",
						}},
					}
					catchBody = append(catchBody, assignment)
				}
				catchBody = append(catchBody, catchStmt.Body.List...)
				break // catch-all должен быть последним
			}
		}
	}

	return &ast.IfStmt{
		If: token.NoPos,
		Cond: &ast.BinaryExpr{
			X: &ast.Ident{
				NamePos: token.NoPos,
				Name:    "err",
			},
			OpPos: token.NoPos,
			Op:    token.NEQ,
			Y: &ast.Ident{
				NamePos: token.NoPos,
				Name:    "nil",
			},
		},
		Body: &ast.BlockStmt{
			Lbrace: token.NoPos,
			List:   catchBody,
			Rbrace: token.NoPos,
		},
	}
}

// makeMultiTypeCheck строит IIFE-условие для нескольких типов:
//
//	func() bool {
//	    if _, ok := err.(T1); ok { return true }
//	    if _, ok := err.(T2); ok { return true }
//	    return false
//	}()
func (t *Transpiler) makeMultiTypeCheck(types []ast.Expr) *ast.CallExpr {
	var body []ast.Stmt
	for _, typ := range types {
		check := &ast.IfStmt{
			If: token.NoPos,
			Init: &ast.AssignStmt{
				Lhs:    []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "_"}, &ast.Ident{NamePos: token.NoPos, Name: "ok"}},
				TokPos: token.NoPos,
				Tok:    token.DEFINE,
				Rhs:    []ast.Expr{&ast.TypeAssertExpr{X: &ast.Ident{NamePos: token.NoPos, Name: "err"}, Type: typ}},
			},
			Cond: &ast.Ident{NamePos: token.NoPos, Name: "ok"},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{&ast.ReturnStmt{
					Return:  token.NoPos,
					Results: []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "true"}},
				}},
			},
		}
		body = append(body, check)
	}
	body = append(body, &ast.ReturnStmt{
		Return:  token.NoPos,
		Results: []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "false"}},
	})

	return &ast.CallExpr{
		Fun: &ast.FuncLit{
			Type: &ast.FuncType{
				Params: &ast.FieldList{},
				Results: &ast.FieldList{
					List: []*ast.Field{{Type: &ast.Ident{NamePos: token.NoPos, Name: "bool"}}},
				},
			},
			Body: &ast.BlockStmt{List: body},
		},
	}
}

// createTypeCheck создает проверку типа ошибки для конкретного catch
func (t *Transpiler) createTypeCheck(catchStmt *ast.CatchStmt, isLast bool) ast.Stmt {
	var condition ast.Expr
	var init ast.Stmt
	var bodyPrefix []ast.Stmt

	if len(catchStmt.ErrorTypes) > 1 {
		// Multi-type: условие через IIFE
		condition = t.makeMultiTypeCheck(catchStmt.ErrorTypes)
		if catchStmt.ErrorVar != nil {
			bodyPrefix = []ast.Stmt{&ast.AssignStmt{
				Lhs:    []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: catchStmt.ErrorVar.Name}},
				TokPos: token.NoPos,
				Tok:    token.DEFINE,
				Rhs:    []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "err"}},
			}}
		}
	} else {
		typ := catchStmt.ErrorTypes[0]
		if catchStmt.ErrorVar != nil {
			// catchVar, ok := err.(ErrorType)
			init = &ast.AssignStmt{
				Lhs:    []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: catchStmt.ErrorVar.Name}, &ast.Ident{NamePos: token.NoPos, Name: "ok"}},
				TokPos: token.NoPos,
				Tok:    token.DEFINE,
				Rhs:    []ast.Expr{&ast.TypeAssertExpr{X: &ast.Ident{NamePos: token.NoPos, Name: "err"}, Type: typ}},
			}
		} else {
			// _, ok := err.(ErrorType)
			init = &ast.AssignStmt{
				Lhs:    []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "_"}, &ast.Ident{NamePos: token.NoPos, Name: "ok"}},
				TokPos: token.NoPos,
				Tok:    token.DEFINE,
				Rhs:    []ast.Expr{&ast.TypeAssertExpr{X: &ast.Ident{NamePos: token.NoPos, Name: "err"}, Type: typ}},
			}
		}
		condition = &ast.Ident{NamePos: token.NoPos, Name: "ok"}
	}

	body := append(bodyPrefix, catchStmt.Body.List...)

	ifStmt := &ast.IfStmt{
		If:   token.NoPos,
		Init: init,
		Cond: condition,
		Body: &ast.BlockStmt{Lbrace: token.NoPos, List: body, Rbrace: token.NoPos},
	}

	if !isLast {
		ifStmt.Else = &ast.BlockStmt{
			Lbrace: token.NoPos,
			List:   []ast.Stmt{&ast.ReturnStmt{Return: token.NoPos, Results: []ast.Expr{&ast.Ident{NamePos: token.NoPos, Name: "err"}}}},
			Rbrace: token.NoPos,
		}
	}

	return ifStmt
}

// TranspileFile главная функция для транспиляции
func TranspileFile(source string) (string, error) {
	transpiler := NewTranspiler()
	return transpiler.Transpile(source)
}

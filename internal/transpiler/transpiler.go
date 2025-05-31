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

// transpileStmts транспилирует список statements
func (t *Transpiler) transpileStmts(stmts []ast.Stmt) []ast.Stmt {
	var result []ast.Stmt

	for _, stmt := range stmts {
		if tryStmt, ok := stmt.(*ast.TryStmt); ok {
			// Транспилируем try-catch в обычные конструкции
			transpiled := t.transpileTryStmt(tryStmt)
			result = append(result, transpiled...)
		} else {
			// Рекурсивно обрабатываем вложенные блоки
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
			Cond: s.Cond,
			Body: &ast.BlockStmt{
				List: t.transpileStmts(s.Body.List),
			},
			Else: s.Else,
		}
	case *ast.ForStmt:
		return &ast.ForStmt{
			For:  s.For,
			Init: s.Init,
			Cond: s.Cond,
			Post: s.Post,
			Body: &ast.BlockStmt{
				List: t.transpileStmts(s.Body.List),
			},
		}
	default:
		return stmt
	}
}

// transpileTryStmt транспилирует TryStmt в обычные Go конструкции
func (t *Transpiler) transpileTryStmt(tryStmt *ast.TryStmt) []ast.Stmt {
	var result []ast.Stmt

	// Транспилируем содержимое try блока
	for _, stmt := range tryStmt.Body.List {
		result = append(result, stmt)

		// Добавляем проверку err != nil только если есть комментарий //@errcheck
		if t.hasErrCheckComment(stmt) {
			errorCheck := t.createErrorCheck(tryStmt.Catches)
			result = append(result, errorCheck)
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
			if catchStmt.ErrorType != nil {
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

// createTypeCheck создает проверку типа ошибки для конкретного catch
func (t *Transpiler) createTypeCheck(catchStmt *ast.CatchStmt, isLast bool) ast.Stmt {
	var condition ast.Expr
	var init ast.Stmt

	if catchStmt.ErrorVar != nil {
		// catchVar, ok := err.(ErrorType)
		init = &ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.Ident{
					NamePos: token.NoPos,
					Name:    catchStmt.ErrorVar.Name,
				},
				&ast.Ident{
					NamePos: token.NoPos,
					Name:    "ok",
				},
			},
			TokPos: token.NoPos,
			Tok:    token.DEFINE,
			Rhs: []ast.Expr{
				&ast.TypeAssertExpr{
					X:      &ast.Ident{NamePos: token.NoPos, Name: "err"},
					Lparen: token.NoPos,
					Type:   catchStmt.ErrorType,
					Rparen: token.NoPos,
				},
			},
		}
		condition = &ast.Ident{NamePos: token.NoPos, Name: "ok"}
	} else {
		// _, ok := err.(ErrorType)
		init = &ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.Ident{NamePos: token.NoPos, Name: "_"},
				&ast.Ident{NamePos: token.NoPos, Name: "ok"},
			},
			TokPos: token.NoPos,
			Tok:    token.DEFINE,
			Rhs: []ast.Expr{
				&ast.TypeAssertExpr{
					X:      &ast.Ident{NamePos: token.NoPos, Name: "err"},
					Lparen: token.NoPos,
					Type:   catchStmt.ErrorType,
					Rparen: token.NoPos,
				},
			},
		}
		condition = &ast.Ident{NamePos: token.NoPos, Name: "ok"}
	}

	ifStmt := &ast.IfStmt{
		If:   token.NoPos,
		Init: init,
		Cond: condition,
		Body: &ast.BlockStmt{
			Lbrace: token.NoPos,
			List:   catchStmt.Body.List,
			Rbrace: token.NoPos,
		},
	}

	// Если это не последний catch и нет else, добавляем return err
	if !isLast {
		ifStmt.Else = &ast.BlockStmt{
			Lbrace: token.NoPos,
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Return: token.NoPos,
					Results: []ast.Expr{
						&ast.Ident{NamePos: token.NoPos, Name: "err"},
					},
				},
			},
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

// internal/transpiler/transpiler.go
package transpiler

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sviridovkonstantin42/godsl/internal/format"

	"github.com/sviridovkonstantin42/godsl/internal/ast"

	"github.com/sviridovkonstantin42/godsl/internal/parser"

	"github.com/sviridovkonstantin42/godsl/internal/token"
)

// TranspileFile транспилирует GoDSL код в стандартный Go
func TranspileFile(source string) (string, error) {
	transpiler := NewTranspiler()
	return transpiler.Transpile(source)
}

// Transpiler преобразует GoDSL в Go код
type Transpiler struct {
	fset          *token.FileSet
	currentTry    *TryContext
	tryStack      []*TryContext
	checkErrRegex *regexp.Regexp
}

// TryContext содержит информацию о текущем try блоке
type TryContext struct {
	catchBlock *ast.BlockStmt
	depth      int
}

// NewTranspiler создает новый транспилятор
func NewTranspiler() *Transpiler {
	return &Transpiler{
		fset:          token.NewFileSet(),
		tryStack:      make([]*TryContext, 0),
		checkErrRegex: regexp.MustCompile(`//\s*checkerr\s*$`),
	}
}

// Transpile выполняет транспиляцию исходного кода
func (t *Transpiler) Transpile(source string) (string, error) {
	// Предварительная обработка try-catch блоков
	processedSource, err := t.preprocessTryCatch(source)
	if err != nil {
		return "", fmt.Errorf("preprocessing error: %v", err)
	}

	// Парсим обработанный код
	file, err := parser.ParseFile(t.fset, "", processedSource, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("parsing error: %v", err)
	}

	// Трансформируем AST
	t.transformAST(file)

	// Генерируем финальный код
	return t.generateCode(file)
}

// preprocessTryCatch обрабатывает try-catch блоки в исходном коде
func (t *Transpiler) preprocessTryCatch(source string) (string, error) {
	lines := strings.Split(source, "\n")
	result := make([]string, 0, len(lines))

	tryStack := make([]TryBlock, 0)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Обработка try блоков
		if strings.HasPrefix(trimmed, "try {") || trimmed == "try {" {
			tryBlock := TryBlock{
				StartLine: i,
				Depth:     len(tryStack),
				Lines:     make([]string, 0),
			}
			tryStack = append(tryStack, tryBlock)
			result = append(result, "{ // TRY_START")
			continue
		}

		// Обработка catch блоков
		if strings.HasPrefix(trimmed, "catch {") || trimmed == "catch {" {
			if len(tryStack) == 0 {
				return "", fmt.Errorf("catch without try at line %d", i+1)
			}

			currentTry := &tryStack[len(tryStack)-1]
			currentTry.HasCatch = true
			currentTry.CatchStart = i
			result = append(result, "{ // CATCH_START")
			continue
		}

		// Обработка закрывающих скобок
		if trimmed == "}" && len(tryStack) > 0 {
			currentTry := &tryStack[len(tryStack)-1]
			if !currentTry.HasCatch {
				// Конец try блока без catch
				result = append(result, "} // TRY_END_NO_CATCH")
				tryStack = tryStack[:len(tryStack)-1]
			} else if currentTry.CatchStart > 0 && len(currentTry.CatchLines) == 0 {
				// Начинаем собирать строки catch блока
				currentTry.CatchLines = make([]string, 0)
				result = append(result, "} // CATCH_END")
				tryStack = tryStack[:len(tryStack)-1]
			} else {
				result = append(result, line)
			}
			continue
		}

		result = append(result, line)
	}

	if len(tryStack) > 0 {
		return "", fmt.Errorf("unclosed try block")
	}

	return strings.Join(result, "\n"), nil
}

// TryBlock представляет try блок в исходном коде
type TryBlock struct {
	StartLine  int
	Depth      int
	Lines      []string
	HasCatch   bool
	CatchStart int
	CatchLines []string
}

// transformAST трансформирует AST для обработки //checkerr комментариев
func (t *Transpiler) transformAST(file *ast.File) {
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.BlockStmt:
			t.transformBlockStmt(n)
		}
		return true
	})
}

// transformBlockStmt обрабатывает блок statements
func (t *Transpiler) transformBlockStmt(block *ast.BlockStmt) {
	newStmts := make([]ast.Stmt, 0, len(block.List)*2)

	for _, stmt := range block.List {
		newStmts = append(newStmts, stmt)

		// Проверяем комментарии после statement
		// if t.hasCheckErrComment(stmt, block, i) {
		// 	errorCheck := t.generateErrorCheck()
		// 	newStmts = append(newStmts, errorCheck)
		// }
	}

	block.List = newStmts
}

// hasCheckErrComment проверяет наличие комментария //checkerr
// func (t *Transpiler) hasCheckErrComment(stmt ast.Stmt, block *ast.BlockStmt, index int) bool {
// 	// Получаем позицию statement
// 	stmtPos := t.fset.Position(stmt.Pos())
// 	stmtEnd := t.fset.Position(stmt.End())

// 	// Ищем комментарий на той же строке
// 	for _, commentGroup := range t.fset.Comments {
// 		if commentGroup == nil {
// 			continue
// 		}

// 		for _, comment := range commentGroup.List {
// 			commentPos := t.fset.Position(comment.Pos())

// 			// Проверяем, что комментарий на той же строке
// 			if commentPos.Line == stmtEnd.Line {
// 				return t.checkErrRegex.MatchString(comment.Text)
// 			}
// 		}
// 	}

// 	return false
// }

// generateErrorCheck создает проверку if err != nil
func (t *Transpiler) generateErrorCheck() ast.Stmt {
	// Создаем условие: err != nil
	condition := &ast.BinaryExpr{
		X:  &ast.Ident{Name: "err"},
		Op: token.NEQ,
		Y:  &ast.Ident{Name: "nil"},
	}

	// Создаем тело if блока
	var body *ast.BlockStmt

	if t.currentTry != nil && t.currentTry.catchBlock != nil {
		// Используем код из catch блока
		body = t.currentTry.catchBlock
	} else {
		// Стандартная обработка - return err
		body = &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.Ident{Name: "err"},
					},
				},
			},
		}
	}

	return &ast.IfStmt{
		Cond: condition,
		Body: body,
	}
}

// generateCode генерирует финальный Go код
func (t *Transpiler) generateCode(file *ast.File) (string, error) {
	var buf strings.Builder

	err := format.Node(&buf, t.fset, file)
	if err != nil {
		return "", fmt.Errorf("code generation error: %v", err)
	}

	code := buf.String()

	// Постобработка сгенерированного кода
	code = t.postprocessCode(code)

	return code, nil
}

// postprocessCode выполняет финальную обработку кода
func (t *Transpiler) postprocessCode(code string) string {
	// Убираем служебные комментарии
	code = strings.ReplaceAll(code, "// TRY_START", "")
	code = strings.ReplaceAll(code, "// CATCH_START", "")
	code = strings.ReplaceAll(code, "// TRY_END_NO_CATCH", "")
	code = strings.ReplaceAll(code, "// CATCH_END", "")

	// Очищаем лишние пустые строки
	lines := strings.Split(code, "\n")
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) != "" || len(result) == 0 || strings.TrimSpace(result[len(result)-1]) != "" {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// Дополнительные утилиты для работы с AST

// findErrorVariable находит переменную ошибки в assignment
func (t *Transpiler) findErrorVariable(stmt ast.Stmt) string {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		for _, lhs := range s.Lhs {
			if ident, ok := lhs.(*ast.Ident); ok {
				if ident.Name == "err" {
					return "err"
				}
			}
		}
	}
	return "err" // по умолчанию
}

// isErrorReturningCall проверяет, возвращает ли вызов ошибку
func (t *Transpiler) isErrorReturningCall(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.CallExpr:
		return true // предполагаем, что все вызовы могут вернуть ошибку
	default:
		_ = e
		return false
	}
}

// createReturnStmt создает return statement с ошибкой
func (t *Transpiler) createReturnStmt(errorVar string) *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Results: []ast.Expr{
			&ast.Ident{Name: errorVar},
		},
	}
}

// Enhanced version with better comment handling
type CommentProcessor struct {
	fset     *token.FileSet
	comments []*ast.CommentGroup
}

func (cp *CommentProcessor) findCommentForLine(line int) *ast.Comment {
	for _, group := range cp.comments {
		for _, comment := range group.List {
			pos := cp.fset.Position(comment.Pos())
			if pos.Line == line {
				return comment
			}
		}
	}
	return nil
}

// Улучшенная версия парсера с обработкой комментариев
func (t *Transpiler) parseWithComments(source string) (*ast.File, error) {
	file, err := parser.ParseFile(t.fset, "", source, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// // Сохраняем комментарии для дальнейшей обработки
	// t.fset.Comments = file.Comments

	return file, nil
}

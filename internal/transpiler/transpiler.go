package transpiler

import (
	"fmt"
	"strings"

	"github.com/sviridovkonstantin42/godsl/internal/lexer"
	"github.com/sviridovkonstantin42/godsl/internal/parser"
)

// Transpiler структура для транспиляции GoDSL в Go
type Transpiler struct {
	errorHandlingEnabled bool
	currentFunction      string
	errorVarCounter      int
	indentLevel          int
}

// New создает новый транспилятор
func New() *Transpiler {
	return &Transpiler{
		errorHandlingEnabled: true,
		errorVarCounter:      0,
		indentLevel:          0,
	}
}

// TranspileProgram транспилирует AST программы в Go код
func (t *Transpiler) TranspileProgram(program *parser.Program) string {
	var result strings.Builder

	// Добавляем стандартные импорты для обработки ошибок если нужно
	needsErrorHandling := t.programNeedsErrorHandling(program)
	if needsErrorHandling {
		result.WriteString("import (\n")
		result.WriteString("\t\"errors\"\n")
		result.WriteString("\t\"fmt\"\n")
		result.WriteString(")\n\n")
	}

	for i, stmt := range program.Statements {
		transpiled := t.transpileStatement(stmt)
		result.WriteString(transpiled)

		// Добавляем перенос строки между statements, кроме последнего
		if i < len(program.Statements)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// programNeedsErrorHandling проверяет, нужна ли обработка ошибок
func (t *Transpiler) programNeedsErrorHandling(program *parser.Program) bool {
	for _, stmt := range program.Statements {
		if t.statementNeedsErrorHandling(stmt) {
			return true
		}
	}
	return false
}

func (t *Transpiler) statementNeedsErrorHandling(stmt parser.Statement) bool {
	switch s := stmt.(type) {
	case *parser.TryStatement:
		return true
	case *parser.ThrowStatement:
		return true
	case *parser.FunctionStatement:
		return t.blockNeedsErrorHandling(s.Body)
	case *parser.BlockStatement:
		return t.blockNeedsErrorHandling(s)
	case *parser.IfStatement:
		return t.blockNeedsErrorHandling(s.Consequence) ||
			(s.Alternative != nil && t.blockNeedsErrorHandling(s.Alternative))
	case *parser.ForStatement:
		return t.blockNeedsErrorHandling(s.Body)
	}
	return false
}

func (t *Transpiler) blockNeedsErrorHandling(block *parser.BlockStatement) bool {
	if block == nil {
		return false
	}
	for _, stmt := range block.Statements {
		if t.statementNeedsErrorHandling(stmt) {
			return true
		}
	}
	return false
}

// transpileStatement транспилирует statement
func (t *Transpiler) transpileStatement(stmt parser.Statement) string {
	switch s := stmt.(type) {
	case *parser.PackageStatement:
		return t.transpilePackageStatement(s)
	case *parser.ImportStatement:
		return t.transpileImportStatement(s)
	case *parser.VarStatement:
		return t.transpileVarStatement(s)
	case *parser.ConstStatement:
		return t.transpileConstStatement(s)
	case *parser.FunctionStatement:
		return t.transpileFunctionStatement(s)
	case *parser.ReturnStatement:
		return t.transpileReturnStatement(s)
	case *parser.IfStatement:
		return t.transpileIfStatement(s)
	case *parser.ForStatement:
		return t.transpileForStatement(s)
	case *parser.TryStatement:
		return t.transpileTryStatement(s)
	case *parser.ThrowStatement:
		return t.transpileThrowStatement(s)
	case *parser.ExpressionStatement:
		return t.transpileExpressionStatement(s)
	case *parser.BlockStatement:
		return t.transpileBlockStatement(s)
	case *parser.AssignStatement:
		return t.transpileAssignStatement(s)
	case *parser.CommentStatement:
		return t.transpileCommentStatement(s)
	case *parser.IncrementStatement:
		return t.transpileIncrementStatement(s)
	case *parser.DefineStatement:
		return t.transpileDefineStatement(s)
	default:
		return fmt.Sprintf("// Unknown statement type: %T", s)
	}
}

func (t *Transpiler) transpileCommentStatement(stmt *parser.CommentStatement) string {
	if stmt.IsMultiLine {
		return t.indent() + "/*" + stmt.Value + "*/"
	}

	return t.indent() + "// " + stmt.Value
}

func (t *Transpiler) transpileIncrementStatement(expr *parser.IncrementStatement) string {
	return t.indent() + expr.String()
}

func (t *Transpiler) transpilePackageStatement(stmt *parser.PackageStatement) string {
	return fmt.Sprintf("package %s", stmt.Value)
}

func (t *Transpiler) transpileImportStatement(stmt *parser.ImportStatement) string {
	if len(stmt.Imports) == 1 {
		return fmt.Sprintf("import \"%s\"", stmt.Imports[0])
	}

	var result strings.Builder
	result.WriteString("import (\n")
	for _, imp := range stmt.Imports {
		result.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
	}
	result.WriteString(")")
	return result.String()
}

func (t *Transpiler) transpileVarStatement(stmt *parser.VarStatement) string {
	var result strings.Builder
	result.WriteString(t.indent() + "var " + stmt.Name.Value)

	if stmt.Type != "" {
		result.WriteString(" " + stmt.Type)
	}

	if stmt.Value != nil {
		result.WriteString(" = " + t.transpileExpression(stmt.Value))
	}

	return result.String()
}

func (t *Transpiler) transpileConstStatement(stmt *parser.ConstStatement) string {
	var result strings.Builder
	result.WriteString(t.indent() + "const " + stmt.Name.Value)

	if stmt.Type != "" {
		result.WriteString(" " + stmt.Type)
	}

	result.WriteString(" = " + t.transpileExpression(stmt.Value))
	return result.String()
}

func (t *Transpiler) transpileFunctionStatement(stmt *parser.FunctionStatement) string {
	t.currentFunction = stmt.Name.Value

	var result strings.Builder
	result.WriteString(t.indent() + "func " + stmt.Name.Value + "(")

	// Параметры
	for i, param := range stmt.Parameters {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(param.Value)
		if i < len(stmt.ParamTypes) && stmt.ParamTypes[i] != "" {
			result.WriteString(" " + stmt.ParamTypes[i])
		}
	}
	result.WriteString(")")

	// Возвращаемый тип
	if stmt.ReturnType != "" {
		// Если функция содержит try-catch, добавляем error к возвращаемому типу
		if t.blockNeedsErrorHandling(stmt.Body) {
			if stmt.ReturnType == "error" {
				result.WriteString(" error")
			} else {
				result.WriteString(" (" + stmt.ReturnType + ", error)")
			}
		} else {
			result.WriteString(" " + stmt.ReturnType)
		}
	} else if t.blockNeedsErrorHandling(stmt.Body) {
		result.WriteString(" error")
	}

	result.WriteString(" ")
	result.WriteString(t.transpileBlockStatement(stmt.Body))

	return result.String()
}

func (t *Transpiler) transpileBlockStatement(stmt *parser.BlockStatement) string {
	var result strings.Builder
	result.WriteString("{\n")

	t.indentLevel++
	for _, s := range stmt.Statements {
		transpiled := t.transpileStatement(s)
		if transpiled != "" {
			result.WriteString(transpiled)
			if !strings.HasSuffix(transpiled, "\n") {
				result.WriteString("\n")
			}
		}
	}
	t.indentLevel--

	result.WriteString(t.indent() + "}")
	return result.String()
}

func (t *Transpiler) transpileReturnStatement(stmt *parser.ReturnStatement) string {
	if stmt.ReturnValue != nil {
		return t.indent() + "return " + t.transpileExpression(stmt.ReturnValue)
	}
	return t.indent() + "return"
}

func (t *Transpiler) transpileIfStatement(stmt *parser.IfStatement) string {
	var result strings.Builder
	result.WriteString(t.indent() + "if " + t.transpileExpression(stmt.Condition) + " ")
	result.WriteString(t.transpileBlockStatement(stmt.Consequence))

	if stmt.Alternative != nil {
		result.WriteString(" else ")
		result.WriteString(t.transpileBlockStatement(stmt.Alternative))
	}

	return result.String()
}

func (t *Transpiler) transpileForStatement(stmt *parser.ForStatement) string {
	var result strings.Builder
	result.WriteString(t.indent() + "for ")

	if stmt.Init != nil {
		result.WriteString(strings.TrimSpace(t.transpileStatement(stmt.Init)))
	}
	result.WriteString("; ")

	if stmt.Condition != nil {
		result.WriteString(t.transpileExpression(stmt.Condition))
	}
	result.WriteString("; ")

	if stmt.Update != nil {
		result.WriteString(strings.TrimSpace(t.transpileStatement(stmt.Update)))
	}

	result.WriteString(" ")
	result.WriteString(t.transpileBlockStatement(stmt.Body))

	return result.String()
}

func (t *Transpiler) transpileTryStatement(stmt *parser.TryStatement) string {
	var result strings.Builder

	// // Генерируем функцию-обертку для try-catch
	// funcName := fmt.Sprintf("tryBlock%d", t.errorVarCounter)
	t.errorVarCounter++

	// Создаем анонимную функцию для try блока
	result.WriteString(t.indent() + "err := func() error {\n")
	t.indentLevel++

	// Транспилируем содержимое try блока
	for _, s := range stmt.Body.Statements {
		transpiled := t.transpileStatement(s)
		if transpiled != "" {
			result.WriteString(transpiled)
			if !strings.HasSuffix(transpiled, "\n") {
				result.WriteString("\n")
			}
		}
	}

	result.WriteString(t.indent() + "return nil\n")
	t.indentLevel--
	result.WriteString(t.indent() + "}()\n")

	// Генерируем catch блоки
	if len(stmt.CatchBlocks) > 0 {
		result.WriteString(t.indent() + "if err != nil {\n")
		t.indentLevel++

		for i, catchBlock := range stmt.CatchBlocks {
			if i > 0 {
				result.WriteString(t.indent() + "} else ")
			}

			// Если есть тип исключения, проверяем его
			if catchBlock.Type != "" {
				result.WriteString(fmt.Sprintf("if _, ok := err.(*%s); ok ", catchBlock.Type))
			}

			result.WriteString("{\n")
			t.indentLevel++

			// Если есть переменная для исключения, присваиваем ей значение
			if catchBlock.Exception != nil {
				result.WriteString(t.indent() + fmt.Sprintf("%s := err\n", catchBlock.Exception.Value))
			}

			// Транспилируем тело catch блока
			for _, s := range catchBlock.Body.Statements {
				transpiled := t.transpileStatement(s)
				if transpiled != "" {
					result.WriteString(transpiled)
					if !strings.HasSuffix(transpiled, "\n") {
						result.WriteString("\n")
					}
				}
			}

			t.indentLevel--
		}

		result.WriteString(t.indent() + "}\n")
		t.indentLevel--
		result.WriteString(t.indent() + "}\n")
	}

	// Finally блок
	if stmt.Finally != nil {
		result.WriteString(t.indent() + "// Finally block\n")
		for _, s := range stmt.Finally.Statements {
			transpiled := t.transpileStatement(s)
			if transpiled != "" {
				result.WriteString(transpiled)
				if !strings.HasSuffix(transpiled, "\n") {
					result.WriteString("\n")
				}
			}
		}
	}

	return result.String()
}

func (t *Transpiler) transpileThrowStatement(stmt *parser.ThrowStatement) string {
	expr := t.transpileExpression(stmt.Value)

	// Если выражение не является error, оборачиваем его
	return t.indent() + fmt.Sprintf("return errors.New(fmt.Sprintf(\"%%v\", %s))", expr)
}

func (t *Transpiler) transpileExpressionStatement(stmt *parser.ExpressionStatement) string {
	return t.indent() + t.transpileExpression(stmt.Expression)
}

func (t *Transpiler) transpileAssignStatement(stmt *parser.AssignStatement) string {
	return t.indent() + fmt.Sprintf("%s = %s", stmt.Name.Value, t.transpileExpression(stmt.Value))
}

func (t *Transpiler) transpileDefineStatement(stmt *parser.DefineStatement) string {
	return t.indent() + fmt.Sprintf("%s := %s", stmt.Name.Value, t.transpileExpression(stmt.Value))
}

// transpileExpression транспилирует выражения
func (t *Transpiler) transpileExpression(expr parser.Expression) string {
	switch e := expr.(type) {
	case *parser.Identifier:
		return e.Value
	case *parser.IntegerLiteral:
		return fmt.Sprintf("%d", e.Value)
	case *parser.FloatLiteral:
		return fmt.Sprintf("%f", e.Value)
	case *parser.StringLiteral:
		return fmt.Sprintf("\"%s\"", e.Value)
	case *parser.BooleanLiteral:
		return fmt.Sprintf("%t", e.Value)
	case *parser.PrefixExpression:
		return fmt.Sprintf("%s%s", e.Operator, t.transpileExpression(e.Right))
	case *parser.InfixExpression:
		return fmt.Sprintf("%s %s %s",
			t.transpileExpression(e.Left),
			e.Operator,
			t.transpileExpression(e.Right))
	case *parser.DotExpression:
		return fmt.Sprintf("%s.%s", t.transpileExpression(e.Left), e.Property.Value)
	case *parser.CallExpression:
		var args []string
		for _, arg := range e.Arguments {
			args = append(args, t.transpileExpression(arg))
		}
		return fmt.Sprintf("%s(%s)",
			t.transpileExpression(e.Function),
			strings.Join(args, ", "))
	default:
		return fmt.Sprintf("/* Unknown expression type: %T */", e)
	}
}

func (t *Transpiler) indent() string {
	return strings.Repeat("\t", t.indentLevel)
}

func TranspileFile(input string) (string, error) {
	lexer := lexer.New(input)
	parser := parser.New(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString("Parser errors:\n")
		for _, err := range parser.Errors() {
			errorMsg.WriteString(fmt.Sprintf("  %s\n", err))
		}
		return "", fmt.Errorf("%s", errorMsg.String())
	}

	transpiler := New()
	result := transpiler.TranspileProgram(program)

	return result, nil
}

// CustomErrorType представляет пользовательский тип ошибки
type CustomErrorType struct {
	Name    string
	Message string
}

func (e *CustomErrorType) Error() string {
	return fmt.Sprintf("%s: %s", e.Name, e.Message)
}

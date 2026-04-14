package transpiler

import (
	"bytes"
	"fmt"

	"github.com/sviridovkonstantin42/godsl/internal/format"
	"github.com/sviridovkonstantin42/godsl/internal/parser"
	"github.com/sviridovkonstantin42/godsl/internal/token"
)

// FormatFile форматирует .godsl исходный код без транспиляции.
// Парсит файл, затем печатает AST обратно в .godsl синтаксис.
func FormatFile(source string) (string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("parse error: %v", err)
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return "", fmt.Errorf("format error: %v", err)
	}

	return buf.String(), nil
}

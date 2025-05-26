package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sviridovkonstantin42/godsl/internal/lexer"
)

// AST Node интерфейсы
type Node interface {
	String() string
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

// Program - корневой узел AST
type Program struct {
	Statements []Statement
}

func (p *Program) String() string {
	var out strings.Builder
	for _, s := range p.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

// Statements
type PackageStatement struct {
	Token lexer.Token
	Value string
}

func (ps *PackageStatement) statementNode() {}
func (ps *PackageStatement) String() string {
	return fmt.Sprintf("package %s", ps.Value)
}

type ImportStatement struct {
	Token   lexer.Token
	Imports []string
}

func (is *ImportStatement) statementNode() {}
func (is *ImportStatement) String() string {
	if len(is.Imports) == 1 {
		return fmt.Sprintf("import %s", is.Imports[0])
	}
	return fmt.Sprintf("import (%s)", strings.Join(is.Imports, ", "))
}

type VarStatement struct {
	Token lexer.Token
	Name  *Identifier
	Type  string
	Value Expression
}

func (vs *VarStatement) statementNode() {}
func (vs *VarStatement) String() string {
	var out strings.Builder
	out.WriteString("var ")
	out.WriteString(vs.Name.String())
	if vs.Type != "" {
		out.WriteString(" ")
		out.WriteString(vs.Type)
	}
	if vs.Value != nil {
		out.WriteString(" = ")
		out.WriteString(vs.Value.String())
	}
	return out.String()
}

type ConstStatement struct {
	Token lexer.Token
	Name  *Identifier
	Type  string
	Value Expression
}

func (cs *ConstStatement) statementNode() {}
func (cs *ConstStatement) String() string {
	var out strings.Builder
	out.WriteString("const ")
	out.WriteString(cs.Name.String())
	if cs.Type != "" {
		out.WriteString(" ")
		out.WriteString(cs.Type)
	}
	if cs.Value != nil {
		out.WriteString(" = ")
		out.WriteString(cs.Value.String())
	}
	return out.String()
}

type AssignStatement struct {
	Token lexer.Token
	Name  *Identifier
	Value Expression
}

func (as *AssignStatement) statementNode() {}
func (as *AssignStatement) String() string {
	return fmt.Sprintf("%s = %s", as.Name.String(), as.Value.String())
}

type DefineStatement struct {
	Token lexer.Token
	Name  *Identifier
	Value Expression
}

func (ds *DefineStatement) statementNode() {}
func (ds *DefineStatement) String() string {
	return fmt.Sprintf("%s := %s", ds.Name.String(), ds.Value.String())
}

type ReturnStatement struct {
	Token       lexer.Token
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode() {}
func (rs *ReturnStatement) String() string {
	if rs.ReturnValue != nil {
		return fmt.Sprintf("return %s", rs.ReturnValue.String())
	}
	return "return"
}

type ExpressionStatement struct {
	Token      lexer.Token
	Expression Expression
}

func (es *ExpressionStatement) statementNode() {}
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

type BlockStatement struct {
	Token      lexer.Token
	Statements []Statement
}

func (bs *BlockStatement) statementNode() {}
func (bs *BlockStatement) String() string {
	var out strings.Builder
	out.WriteString("{\n")
	for _, s := range bs.Statements {
		out.WriteString(s.String())
		out.WriteString("\n")
	}
	out.WriteString("}")
	return out.String()
}

type IfStatement struct {
	Token       lexer.Token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (ifs *IfStatement) statementNode() {}
func (ifs *IfStatement) String() string {
	var out strings.Builder
	out.WriteString("if ")
	out.WriteString(ifs.Condition.String())
	out.WriteString(" ")
	out.WriteString(ifs.Consequence.String())
	if ifs.Alternative != nil {
		out.WriteString(" else ")
		out.WriteString(ifs.Alternative.String())
	}
	return out.String()
}

type ForStatement struct {
	Token     lexer.Token
	Init      Statement
	Condition Expression
	Update    Statement
	Body      *BlockStatement
}

func (fs *ForStatement) statementNode() {}
func (fs *ForStatement) String() string {
	var out strings.Builder
	out.WriteString("for ")
	if fs.Init != nil {
		out.WriteString(fs.Init.String())
	}
	out.WriteString("; ")
	if fs.Condition != nil {
		out.WriteString(fs.Condition.String())
	}
	out.WriteString("; ")
	if fs.Update != nil {
		out.WriteString(fs.Update.String())
	}
	out.WriteString(" ")
	out.WriteString(fs.Body.String())
	return out.String()
}

type FunctionStatement struct {
	Token      lexer.Token
	Name       *Identifier
	Parameters []*Identifier
	ParamTypes []string
	ReturnType string
	Body       *BlockStatement
}

func (fs *FunctionStatement) statementNode() {}
func (fs *FunctionStatement) String() string {
	var out strings.Builder
	out.WriteString("func ")
	out.WriteString(fs.Name.String())
	out.WriteString("(")
	for i, param := range fs.Parameters {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(param.String())
		if i < len(fs.ParamTypes) {
			out.WriteString(" ")
			out.WriteString(fs.ParamTypes[i])
		}
	}
	out.WriteString(")")
	if fs.ReturnType != "" {
		out.WriteString(" ")
		out.WriteString(fs.ReturnType)
	}
	out.WriteString(" ")
	out.WriteString(fs.Body.String())
	return out.String()
}

// GoDSL специфичные statements
type TryStatement struct {
	Token       lexer.Token
	Body        *BlockStatement
	CatchBlocks []*CatchBlock
	Finally     *BlockStatement
}

func (ts *TryStatement) statementNode() {}
func (ts *TryStatement) String() string {
	var out strings.Builder
	out.WriteString("try ")
	out.WriteString(ts.Body.String())
	for _, cb := range ts.CatchBlocks {
		out.WriteString(" ")
		out.WriteString(cb.String())
	}
	if ts.Finally != nil {
		out.WriteString(" finally ")
		out.WriteString(ts.Finally.String())
	}
	return out.String()
}

type CatchBlock struct {
	Token     lexer.Token
	Exception *Identifier
	Type      string
	Body      *BlockStatement
}

func (cb *CatchBlock) String() string {
	var out strings.Builder
	out.WriteString("catch")
	if cb.Exception != nil {
		out.WriteString(" (")
		out.WriteString(cb.Exception.String())
		if cb.Type != "" {
			out.WriteString(" ")
			out.WriteString(cb.Type)
		}
		out.WriteString(")")
	}
	out.WriteString(" ")
	out.WriteString(cb.Body.String())
	return out.String()
}

type ThrowStatement struct {
	Token lexer.Token
	Value Expression
}

func (ts *ThrowStatement) statementNode() {}
func (ts *ThrowStatement) String() string {
	return fmt.Sprintf("throw %s", ts.Value.String())
}

// Expressions
type Identifier struct {
	Token lexer.Token
	Value string
}

func (i *Identifier) expressionNode() {}
func (i *Identifier) String() string  { return i.Value }

type IntegerLiteral struct {
	Token lexer.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode() {}
func (il *IntegerLiteral) String() string  { return fmt.Sprintf("%d", il.Value) }

type FloatLiteral struct {
	Token lexer.Token
	Value float64
}

func (fl *FloatLiteral) expressionNode() {}
func (fl *FloatLiteral) String() string  { return fmt.Sprintf("%f", fl.Value) }

type StringLiteral struct {
	Token lexer.Token
	Value string
}

func (sl *StringLiteral) expressionNode() {}
func (sl *StringLiteral) String() string  { return fmt.Sprintf("\"%s\"", sl.Value) }

type BooleanLiteral struct {
	Token lexer.Token
	Value bool
}

func (bl *BooleanLiteral) expressionNode() {}
func (bl *BooleanLiteral) String() string  { return fmt.Sprintf("%t", bl.Value) }

type PrefixExpression struct {
	Token    lexer.Token
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode() {}
func (pe *PrefixExpression) String() string {
	return fmt.Sprintf("(%s%s)", pe.Operator, pe.Right.String())
}

type InfixExpression struct {
	Token    lexer.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode() {}
func (ie *InfixExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", ie.Left.String(), ie.Operator, ie.Right.String())
}

type CallExpression struct {
	Token     lexer.Token
	Function  Expression
	Arguments []Expression
}

func (ce *CallExpression) expressionNode() {}
func (ce *CallExpression) String() string {
	var out strings.Builder
	args := make([]string, 0)
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}
	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")
	return out.String()
}

// Parser структура
type Parser struct {
	l *lexer.Lexer

	curToken  lexer.Token
	peekToken lexer.Token

	errors []string

	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
}

type (
	prefixParseFn func() Expression
	infixParseFn  func(Expression) Expression
)

// Приоритеты операторов
const (
	_ int = iota
	LOWEST
	EQUALS      // ==
	LESSGREATER // > или <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X или !X
	CALL        // myFunction(X)
	DOT
)

var precedences = map[lexer.TokenType]int{
	lexer.EQ:       EQUALS,
	lexer.NOT_EQ:   EQUALS,
	lexer.LT:       LESSGREATER,
	lexer.GT:       LESSGREATER,
	lexer.LE:       LESSGREATER,
	lexer.GE:       LESSGREATER,
	lexer.PLUS:     SUM,
	lexer.MINUS:    SUM,
	lexer.SLASH:    PRODUCT,
	lexer.ASTERISK: PRODUCT,
	lexer.PERCENT:  PRODUCT,
	lexer.LPAREN:   CALL,
	lexer.DOT:      DOT,
}

// New создает новый парсер
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	p.prefixParseFns = make(map[lexer.TokenType]prefixParseFn)
	p.registerPrefix(lexer.IDENT, p.parseIdentifier)
	p.registerPrefix(lexer.INT, p.parseIntegerLiteral)
	p.registerPrefix(lexer.FLOAT, p.parseFloatLiteral)
	p.registerPrefix(lexer.STRING, p.parseStringLiteral)
	p.registerPrefix(lexer.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.BANG, p.parsePrefixExpression)
	p.registerPrefix(lexer.MINUS, p.parsePrefixExpression)
	p.registerPrefix(lexer.LPAREN, p.parseGroupedExpression)

	p.infixParseFns = make(map[lexer.TokenType]infixParseFn)
	p.registerInfix(lexer.PLUS, p.parseInfixExpression)
	p.registerInfix(lexer.MINUS, p.parseInfixExpression)
	p.registerInfix(lexer.SLASH, p.parseInfixExpression)
	p.registerInfix(lexer.ASTERISK, p.parseInfixExpression)
	p.registerInfix(lexer.PERCENT, p.parseInfixExpression)
	p.registerInfix(lexer.EQ, p.parseInfixExpression)
	p.registerInfix(lexer.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(lexer.LT, p.parseInfixExpression)
	p.registerInfix(lexer.GT, p.parseInfixExpression)
	p.registerInfix(lexer.LE, p.parseInfixExpression)
	p.registerInfix(lexer.GE, p.parseInfixExpression)
	p.registerInfix(lexer.LPAREN, p.parseCallExpression)
	p.registerInfix(lexer.DOT, p.parseDotExpression)

	// Читаем два токена, чтобы заполнить curToken и peekToken
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) registerPrefix(tokenType lexer.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType lexer.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead",
		t.String(), p.peekToken.Type.String())
	p.errors = append(p.errors, msg)
}

func (p *Parser) noPrefixParseFnError(t lexer.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found", t.String())
	p.errors = append(p.errors, msg)
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

// ParseProgram парсит всю программу
func (p *Parser) ParseProgram() *Program {
	program := &Program{}
	program.Statements = []Statement{}

	for !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() Statement {
	switch p.curToken.Type {
	case lexer.PACKAGE:
		return p.parsePackageStatement()
	case lexer.IMPORT:
		return p.parseImportStatement()
	case lexer.VAR:
		return p.parseVarStatement()
	case lexer.CONST:
		return p.parseConstStatement()
	case lexer.FUNC:
		return p.parseFunctionStatement()
	case lexer.RETURN:
		return p.parseReturnStatement()
	case lexer.IF:
		return p.parseIfStatement()
	case lexer.FOR:
		return p.parseForStatement()
	case lexer.IDENT:
		if p.peekTokenIs(lexer.DEFINE) {
			return p.parseDefineStatement()
		}
		if p.peekTokenIs(lexer.INCREMENT) || p.peekTokenIs(lexer.DECREMENT) {
			return p.parseIncrementStatement()
		}
		return p.parseExpressionStatement()
	case lexer.COMMENT:
		return p.parseCommentStatement(false)
	case lexer.BLOCK_COMMENT:
		return p.parseCommentStatement(true)
	case lexer.TRY:
		return p.parseTryStatement()
	case lexer.THROW:
		return p.parseThrowStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parsePackageStatement() *PackageStatement {
	stmt := &PackageStatement{Token: p.curToken}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	stmt.Value = p.curToken.Literal
	return stmt
}

func (p *Parser) parseImportStatement() *ImportStatement {
	stmt := &ImportStatement{Token: p.curToken}

	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken()
		stmt.Imports = p.parseImportList()
	} else {
		if !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Imports = []string{p.curToken.Literal}
	}

	return stmt
}

func (p *Parser) parseImportList() []string {
	imports := []string{}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return imports
	}

	p.nextToken()

	for {
		if p.curTokenIs(lexer.STRING) {
			imports = append(imports, p.curToken.Literal)
		}

		if !p.peekTokenIs(lexer.COMMA) {
			break
		}

		p.nextToken()
		p.nextToken()
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return imports
}

func (p *Parser) parseVarStatement() *VarStatement {
	stmt := &VarStatement{Token: p.curToken}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Опциональный тип
	if p.peekTokenIs(lexer.IDENT) {
		p.nextToken()
		stmt.Type = p.curToken.Literal
	}

	if p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken()
		p.nextToken()
		stmt.Value = p.parseExpression(LOWEST)
	}

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseConstStatement() *ConstStatement {
	stmt := &ConstStatement{Token: p.curToken}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Опциональный тип
	if p.peekTokenIs(lexer.IDENT) {
		p.nextToken()
		stmt.Type = p.curToken.Literal
	}

	if !p.expectPeek(lexer.ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseFunctionStatement() *FunctionStatement {
	stmt := &FunctionStatement{Token: p.curToken}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	stmt.Parameters, stmt.ParamTypes = p.parseFunctionParameters()

	if p.peekTokenIs(lexer.IDENT) {
		p.nextToken()
		stmt.ReturnType = p.curToken.Literal
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseFunctionParameters() ([]*Identifier, []string) {
	identifiers := []*Identifier{}
	types := []string{}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return identifiers, types
	}

	p.nextToken()

	ident := &Identifier{Token: p.curToken, Value: p.curToken.Literal}
	identifiers = append(identifiers, ident)

	if p.peekTokenIs(lexer.IDENT) {
		p.nextToken()
		types = append(types, p.curToken.Literal)
	}

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		ident := &Identifier{Token: p.curToken, Value: p.curToken.Literal}
		identifiers = append(identifiers, ident)

		if p.peekTokenIs(lexer.IDENT) {
			p.nextToken()
			types = append(types, p.curToken.Literal)
		}
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil, nil
	}

	return identifiers, types
}

func (p *Parser) parseReturnStatement() *ReturnStatement {
	stmt := &ReturnStatement{Token: p.curToken}

	p.nextToken()

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpressionStatement() *ExpressionStatement {
	stmt := &ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseBlockStatement() *BlockStatement {
	block := &BlockStatement{Token: p.curToken}
	block.Statements = []Statement{}

	p.nextToken()

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

func (p *Parser) parseIfStatement() *IfStatement {
	stmt := &IfStatement{Token: p.curToken}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	stmt.Consequence = p.parseBlockStatement()

	if p.peekTokenIs(lexer.ELSE) {
		p.nextToken()

		if !p.expectPeek(lexer.LBRACE) {
			return nil
		}

		stmt.Alternative = p.parseBlockStatement()
	}

	return stmt
}

func (p *Parser) parseForStatement() *ForStatement {
	stmt := &ForStatement{Token: p.curToken}

	p.nextToken()

	// Парсим инициализацию
	if !p.curTokenIs(lexer.SEMICOLON) {
		stmt.Init = p.parseStatement()
	}

	if !p.expectPeek(lexer.SEMICOLON) {
		return nil
	}

	p.nextToken()

	// Парсим условие
	if !p.curTokenIs(lexer.SEMICOLON) {
		stmt.Condition = p.parseExpression(LOWEST)
	}

	if !p.expectPeek(lexer.SEMICOLON) {
		return nil
	}

	p.nextToken()

	// Парсим обновление
	if !p.curTokenIs(lexer.LBRACE) {
		stmt.Update = p.parseStatement()
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseTryStatement() *TryStatement {
	stmt := &TryStatement{Token: p.curToken}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	// Парсим catch блоки
	for p.peekTokenIs(lexer.CATCH) {
		p.nextToken()
		catchBlock := p.parseCatchBlock()
		if catchBlock != nil {
			stmt.CatchBlocks = append(stmt.CatchBlocks, catchBlock)
		}
	}

	// Парсим finally блок
	if p.peekTokenIs(lexer.FINALLY) {
		p.nextToken()
		if !p.expectPeek(lexer.LBRACE) {
			return nil
		}
		stmt.Finally = p.parseBlockStatement()
	}

	return stmt
}

func (p *Parser) parseCatchBlock() *CatchBlock {
	cb := &CatchBlock{Token: p.curToken}

	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken()
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}

		cb.Exception = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

		if p.peekTokenIs(lexer.IDENT) {
			p.nextToken()
			cb.Type = p.curToken.Literal
		}

		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	cb.Body = p.parseBlockStatement()

	return cb
}

func (p *Parser) parseThrowStatement() *ThrowStatement {
	stmt := &ThrowStatement{Token: p.curToken}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// Expression parsing
func (p *Parser) parseExpression(precedence int) Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(lexer.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() Expression {
	return &Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseInfixExpression(left Expression) Expression {
	expression := &InfixExpression{
		Token:    p.curToken,
		Left:     left,
		Operator: p.curToken.Literal,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parseGroupedExpression() Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return exp
}

func (p *Parser) parseCallExpression(fn Expression) Expression {
	exp := &CallExpression{Token: p.curToken, Function: fn}
	exp.Arguments = p.parseExpressionList(lexer.RPAREN)
	return exp
}

func (p *Parser) parseExpressionList(end lexer.TokenType) []Expression {
	args := []Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return args
	}

	p.nextToken()
	args = append(args, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return args
}

func (p *Parser) parseFloatLiteral() Expression {
	lit := &FloatLiteral{Token: p.curToken}

	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as float", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() Expression {
	return &StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBooleanLiteral() Expression {
	return &BooleanLiteral{Token: p.curToken, Value: p.curTokenIs(lexer.TRUE)}
}

func (p *Parser) parsePrefixExpression() Expression {
	expression := &PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)

	return expression
}

func (p *Parser) parseIntegerLiteral() Expression {
	lit := &IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value
	return lit
}

type DotExpression struct {
	Token    lexer.Token // токен DOT
	Left     Expression  // например, "foo"
	Property *Identifier // например, "bar"
}

func (de *DotExpression) expressionNode() {}
func (de *DotExpression) String() string {
	return fmt.Sprintf("(%s.%s)", de.Left.String(), de.Property.String())
}

func (p *Parser) parseDotExpression(left Expression) Expression {
	expr := &DotExpression{
		Token: p.curToken,
		Left:  left,
	}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	expr.Property = &Identifier{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}

	return expr
}

func (p *Parser) parseDefineStatement() Statement {
	stmt := &DefineStatement{Token: p.curToken}

	// Текущий токен - идентификатор (имя переменной)
	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Переходим к токену :=
	p.nextToken()

	// Переходим к значению/выражению после :=
	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

type CommentStatement struct {
	Token       lexer.Token
	Value       string
	IsMultiLine bool
}

func (cs *CommentStatement) statementNode() {}
func (cs *CommentStatement) String() string {
	return fmt.Sprintf("//%s", cs.Value)
}

func (p *Parser) parseCommentStatement(isMultiLine bool) Statement {
	stmt := &CommentStatement{
		Token:       p.curToken,
		IsMultiLine: isMultiLine,
	}

	if isMultiLine {
		// Убираем /* и */ из значения
		value := strings.TrimPrefix(p.curToken.Literal, "/*")
		value = strings.TrimSuffix(value, "*/")
		stmt.Value = strings.TrimSpace(value)
	} else {
		// Убираем // из начала
		stmt.Value = strings.TrimPrefix(p.curToken.Literal, "//")
	}

	return stmt
}

type IncrementStatement struct {
	Token    lexer.Token
	Name     *Identifier
	Operator string
}

func (is *IncrementStatement) statementNode() {}
func (is *IncrementStatement) String() string {
	return fmt.Sprintf("%s%s", is.Name.String(), is.Operator)
}

func (p *Parser) parseIncrementStatement() Statement {
	stmt := &IncrementStatement{Token: p.curToken}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	p.nextToken()

	stmt.Operator = p.curToken.Literal

	return stmt
}

package lexer

import (
	"fmt"
	"strings"
)

// TokenType представляет тип токена
type TokenType int

const (
	// Специальные токены
	ILLEGAL TokenType = iota
	EOF

	// Идентификаторы и литералы
	IDENT  // переменные, функции
	INT    // 123456
	FLOAT  // 123.45
	STRING // "hello"
	CHAR   // 'a'

	// Операторы
	ASSIGN   // =
	PLUS     // +
	MINUS    // -
	BANG     // !
	ASTERISK // *
	SLASH    // /
	PERCENT  // %

	// Сравнение
	LT     // <
	GT     // >
	LE     // <=
	GE     // >=
	EQ     // ==
	NOT_EQ // !=

	// Логические операторы
	AND // &&
	OR  // ||

	// Битовые операторы
	BIT_AND // &
	BIT_OR  // |
	BIT_XOR // ^
	SHL     // <<
	SHR     // >>

	// Составные операторы присваивания
	PLUS_ASSIGN     // +=
	MINUS_ASSIGN    // -=
	MULTIPLY_ASSIGN // *=
	DIVIDE_ASSIGN   // /=
	MODULO_ASSIGN   // %=
	AND_ASSIGN      // &=
	OR_ASSIGN       // |=
	XOR_ASSIGN      // ^=
	SHL_ASSIGN      // <<=
	SHR_ASSIGN      // >>=

	// Инкремент/декремент
	INCREMENT // ++
	DECREMENT // --

	// Разделители
	COMMA     // ,
	SEMICOLON // ;
	COLON     // :
	DOT       // .
	ELLIPSIS  // ...

	// Скобки
	LPAREN   // (
	RPAREN   // )
	LBRACE   // {
	RBRACE   // }
	LBRACKET // [
	RBRACKET // ]

	// Стрелки и каналы
	ARROW      // <-
	DEFINE     // :=
	FUNC_ARROW // ->

	// Ключевые слова Go
	BREAK
	CASE
	CHAN
	CONST
	CONTINUE
	DEFAULT
	DEFER
	ELSE
	FALLTHROUGH
	FOR
	FUNC
	GO
	GOTO
	IF
	IMPORT
	INTERFACE
	MAP
	PACKAGE
	RANGE
	RETURN
	SELECT
	STRUCT
	SWITCH
	TYPE
	VAR

	// Булевы и nil
	TRUE
	FALSE
	NIL

	// Новые токены для GoDSL
	TRY
	CATCH
	THROW
	FINALLY // на будущее

	// Комментарии
	COMMENT
	BLOCK_COMMENT
)

// Token представляет токен с позицией в исходном коде
type Token struct {
	Type     TokenType
	Literal  string
	Position Position
}

// Position представляет позицию в исходном коде
type Position struct {
	Line   int
	Column int
	Offset int
}

// String возвращает строковое представление токена
func (t Token) String() string {
	return fmt.Sprintf("Token{Type: %s, Literal: %q, Position: %v}",
		t.Type.String(), t.Literal, t.Position)
}

// String возвращает строковое представление типа токена
func (tt TokenType) String() string {
	switch tt {
	case ILLEGAL:
		return "ILLEGAL"
	case EOF:
		return "EOF"
	case IDENT:
		return "IDENT"
	case INT:
		return "INT"
	case FLOAT:
		return "FLOAT"
	case STRING:
		return "STRING"
	case CHAR:
		return "CHAR"
	case ASSIGN:
		return "ASSIGN"
	case PLUS:
		return "PLUS"
	case MINUS:
		return "MINUS"
	case BANG:
		return "BANG"
	case ASTERISK:
		return "ASTERISK"
	case SLASH:
		return "SLASH"
	case PERCENT:
		return "PERCENT"
	case LT:
		return "LT"
	case GT:
		return "GT"
	case LE:
		return "LE"
	case GE:
		return "GE"
	case EQ:
		return "EQ"
	case NOT_EQ:
		return "NOT_EQ"
	case AND:
		return "AND"
	case OR:
		return "OR"
	case BIT_AND:
		return "BIT_AND"
	case BIT_OR:
		return "BIT_OR"
	case BIT_XOR:
		return "BIT_XOR"
	case SHL:
		return "SHL"
	case SHR:
		return "SHR"
	case PLUS_ASSIGN:
		return "PLUS_ASSIGN"
	case MINUS_ASSIGN:
		return "MINUS_ASSIGN"
	case MULTIPLY_ASSIGN:
		return "MULTIPLY_ASSIGN"
	case DIVIDE_ASSIGN:
		return "DIVIDE_ASSIGN"
	case MODULO_ASSIGN:
		return "MODULO_ASSIGN"
	case AND_ASSIGN:
		return "AND_ASSIGN"
	case OR_ASSIGN:
		return "OR_ASSIGN"
	case XOR_ASSIGN:
		return "XOR_ASSIGN"
	case SHL_ASSIGN:
		return "SHL_ASSIGN"
	case SHR_ASSIGN:
		return "SHR_ASSIGN"
	case INCREMENT:
		return "INCREMENT"
	case DECREMENT:
		return "DECREMENT"
	case COMMA:
		return "COMMA"
	case SEMICOLON:
		return "SEMICOLON"
	case COLON:
		return "COLON"
	case DOT:
		return "DOT"
	case ELLIPSIS:
		return "ELLIPSIS"
	case LPAREN:
		return "LPAREN"
	case RPAREN:
		return "RPAREN"
	case LBRACE:
		return "LBRACE"
	case RBRACE:
		return "RBRACE"
	case LBRACKET:
		return "LBRACKET"
	case RBRACKET:
		return "RBRACKET"
	case ARROW:
		return "ARROW"
	case DEFINE:
		return "DEFINE"
	case FUNC_ARROW:
		return "FUNC_ARROW"
	case TRY:
		return "TRY"
	case CATCH:
		return "CATCH"
	case THROW:
		return "THROW"
	case FINALLY:
		return "FINALLY"
	case COMMENT:
		return "COMMENT"
	case BLOCK_COMMENT:
		return "BLOCK_COMMENT"
	default:
		if keyword := lookupKeyword(tt); keyword != "" {
			return keyword
		}
		return "UNKNOWN"
	}
}

var keywords = map[string]TokenType{
	"break":       BREAK,
	"case":        CASE,
	"chan":        CHAN,
	"const":       CONST,
	"continue":    CONTINUE,
	"default":     DEFAULT,
	"defer":       DEFER,
	"else":        ELSE,
	"fallthrough": FALLTHROUGH,
	"for":         FOR,
	"func":        FUNC,
	"go":          GO,
	"goto":        GOTO,
	"if":          IF,
	"import":      IMPORT,
	"interface":   INTERFACE,
	"map":         MAP,
	"package":     PACKAGE,
	"range":       RANGE,
	"return":      RETURN,
	"select":      SELECT,
	"struct":      STRUCT,
	"switch":      SWITCH,
	"type":        TYPE,
	"var":         VAR,
	"true":        TRUE,
	"false":       FALSE,
	"nil":         NIL,
	// GoDSL keywords
	"try":     TRY,
	"catch":   CATCH,
	"throw":   THROW,
	"finally": FINALLY,
}

// LookupIdent проверяет, является ли идентификатор ключевым словом
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// lookupKeyword возвращает строковое представление ключевого слова
func lookupKeyword(tokenType TokenType) string {
	for keyword, tt := range keywords {
		if tt == tokenType {
			return strings.ToUpper(keyword)
		}
	}
	return ""
}

// Lexer лексический анализатор
type Lexer struct {
	input        string
	position     int  // текущая позиция в input (указывает на текущий символ)
	readPosition int  // текущая позиция чтения в input (после текущего символа)
	ch           byte // текущий символ
	line         int  // текущая строка
	column       int  // текущая колонка
}

// New создает новый лексер
func New(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

// readChar читает следующий символ и продвигает позицию в input
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // ASCII код для "NUL" - означает конец файла
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++

	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

// peekChar возвращает следующий символ без продвижения позиции
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// peekCharAt возвращает символ на n позиций вперед
func (l *Lexer) peekCharAt(n int) byte {
	pos := l.readPosition + n - 1
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

// currentPosition возвращает текущую позицию
func (l *Lexer) currentPosition() Position {
	return Position{
		Line:   l.line,
		Column: l.column,
		Offset: l.position,
	}
}

// NextToken возвращает следующий токен
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	pos := l.currentPosition()

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: EQ, Literal: string(ch) + string(l.ch), Position: pos}
		} else {
			tok = newToken(ASSIGN, l.ch, pos)
		}
	case '+':
		switch l.peekChar() {
		case '+':
			ch := l.ch
			l.readChar()
			tok = Token{Type: INCREMENT, Literal: string(ch) + string(l.ch), Position: pos}
		case '=':
			ch := l.ch
			l.readChar()
			tok = Token{Type: PLUS_ASSIGN, Literal: string(ch) + string(l.ch), Position: pos}
		default:
			tok = newToken(PLUS, l.ch, pos)
		}
	case '-':
		switch l.peekChar() {
		case '-':
			ch := l.ch
			l.readChar()
			tok = Token{Type: DECREMENT, Literal: string(ch) + string(l.ch), Position: pos}
		case '=':
			ch := l.ch
			l.readChar()
			tok = Token{Type: MINUS_ASSIGN, Literal: string(ch) + string(l.ch), Position: pos}
		case '>':
			ch := l.ch
			l.readChar()
			tok = Token{Type: FUNC_ARROW, Literal: string(ch) + string(l.ch), Position: pos}
		default:
			tok = newToken(MINUS, l.ch, pos)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: NOT_EQ, Literal: string(ch) + string(l.ch), Position: pos}
		} else {
			tok = newToken(BANG, l.ch, pos)
		}
	case '/':
		switch l.peekChar() {
		case '/':
			tok.Type = COMMENT
			tok.Literal = l.readLineComment()
			tok.Position = pos
			return tok
		case '*':
			tok.Type = BLOCK_COMMENT
			tok.Literal = l.readBlockComment()
			tok.Position = pos
			return tok
		case '=':
			ch := l.ch
			l.readChar()
			tok = Token{Type: DIVIDE_ASSIGN, Literal: string(ch) + string(l.ch), Position: pos}
		default:
			tok = newToken(SLASH, l.ch, pos)
		}
	case '*':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: MULTIPLY_ASSIGN, Literal: string(ch) + string(l.ch), Position: pos}
		} else {
			tok = newToken(ASTERISK, l.ch, pos)
		}
	case '%':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: MODULO_ASSIGN, Literal: string(ch) + string(l.ch), Position: pos}
		} else {
			tok = newToken(PERCENT, l.ch, pos)
		}
	case '<':
		switch l.peekChar() {
		case '=':
			ch := l.ch
			l.readChar()
			tok = Token{Type: LE, Literal: string(ch) + string(l.ch), Position: pos}
		case '<':
			ch := l.ch
			l.readChar()
			if l.peekChar() == '=' {
				literal := string(ch) + string(l.ch)
				l.readChar()
				literal += string(l.ch)
				tok = Token{Type: SHL_ASSIGN, Literal: literal, Position: pos}
			} else {
				tok = Token{Type: SHL, Literal: string(ch) + string(l.ch), Position: pos}
			}
		case '-':
			ch := l.ch
			l.readChar()
			tok = Token{Type: ARROW, Literal: string(ch) + string(l.ch), Position: pos}
		default:
			tok = newToken(LT, l.ch, pos)
		}
	case '>':
		switch l.peekChar() {
		case '=':
			ch := l.ch
			l.readChar()
			tok = Token{Type: GE, Literal: string(ch) + string(l.ch), Position: pos}
		case '>':
			ch := l.ch
			l.readChar()
			if l.peekChar() == '=' {
				literal := string(ch) + string(l.ch)
				l.readChar()
				literal += string(l.ch)
				tok = Token{Type: SHR_ASSIGN, Literal: literal, Position: pos}
			} else {
				tok = Token{Type: SHR, Literal: string(ch) + string(l.ch), Position: pos}
			}
		default:
			tok = newToken(GT, l.ch, pos)
		}
	case '&':
		switch l.peekChar() {
		case '&':
			ch := l.ch
			l.readChar()
			tok = Token{Type: AND, Literal: string(ch) + string(l.ch), Position: pos}
		case '=':
			ch := l.ch
			l.readChar()
			tok = Token{Type: AND_ASSIGN, Literal: string(ch) + string(l.ch), Position: pos}
		default:
			tok = newToken(BIT_AND, l.ch, pos)
		}
	case '|':
		switch l.peekChar() {
		case '|':
			ch := l.ch
			l.readChar()
			tok = Token{Type: OR, Literal: string(ch) + string(l.ch), Position: pos}
		case '=':
			ch := l.ch
			l.readChar()
			tok = Token{Type: OR_ASSIGN, Literal: string(ch) + string(l.ch), Position: pos}
		default:
			tok = newToken(BIT_OR, l.ch, pos)
		}
	case '^':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: XOR_ASSIGN, Literal: string(ch) + string(l.ch), Position: pos}
		} else {
			tok = newToken(BIT_XOR, l.ch, pos)
		}
	case ',':
		tok = newToken(COMMA, l.ch, pos)
	case ';':
		tok = newToken(SEMICOLON, l.ch, pos)
	case ':':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: DEFINE, Literal: string(ch) + string(l.ch), Position: pos}
		} else {
			tok = newToken(COLON, l.ch, pos)
		}
	case '.':
		if l.peekChar() == '.' && l.peekCharAt(2) == '.' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			l.readChar()
			literal += string(l.ch)
			tok = Token{Type: ELLIPSIS, Literal: literal, Position: pos}
		} else {
			tok = newToken(DOT, l.ch, pos)
		}
	case '(':
		tok = newToken(LPAREN, l.ch, pos)
	case ')':
		tok = newToken(RPAREN, l.ch, pos)
	case '{':
		tok = newToken(LBRACE, l.ch, pos)
	case '}':
		tok = newToken(RBRACE, l.ch, pos)
	case '[':
		tok = newToken(LBRACKET, l.ch, pos)
	case ']':
		tok = newToken(RBRACKET, l.ch, pos)
	case '"':
		tok.Type = STRING
		tok.Literal = l.readString()
		tok.Position = pos
	case '\'':
		tok.Type = CHAR
		tok.Literal = l.readChar2()
		tok.Position = pos
	case '`':
		tok.Type = STRING
		tok.Literal = l.readRawString()
		tok.Position = pos
	case 0:
		tok.Literal = ""
		tok.Type = EOF
		tok.Position = pos
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)
			tok.Position = pos
			return tok
		} else if isDigit(l.ch) {
			tok.Type, tok.Literal = l.readNumber()
			tok.Position = pos
			return tok
		} else {
			tok = newToken(ILLEGAL, l.ch, pos)
		}
	}

	l.readChar()
	return tok
}

// newToken создает новый токен
func newToken(tokenType TokenType, ch byte, pos Position) Token {
	return Token{Type: tokenType, Literal: string(ch), Position: pos}
}

// readIdentifier читает идентификатор
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber читает число (int или float)
func (l *Lexer) readNumber() (TokenType, string) {
	position := l.position
	tokenType := INT

	for isDigit(l.ch) {
		l.readChar()
	}

	// Проверяем на float
	if l.ch == '.' && isDigit(l.peekChar()) {
		tokenType = FLOAT
		l.readChar() // пропускаем '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	// Поддержка экспоненциальной записи
	if l.ch == 'e' || l.ch == 'E' {
		tokenType = FLOAT
		l.readChar()
		if l.ch == '+' || l.ch == '-' {
			l.readChar()
		}
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return tokenType, l.input[position:l.position]
}

// readString читает строку в двойных кавычках
func (l *Lexer) readString() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
		// Обработка escape последовательностей
		if l.ch == '\\' {
			l.readChar()
		}
	}
	return l.input[position:l.position]
}

// readChar2 читает символ в одинарных кавычках
func (l *Lexer) readChar2() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '\'' || l.ch == 0 {
			break
		}
		// Обработка escape последовательностей
		if l.ch == '\\' {
			l.readChar()
		}
	}
	return l.input[position:l.position]
}

// readRawString читает raw string в обратных кавычках
func (l *Lexer) readRawString() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '`' || l.ch == 0 {
			break
		}
	}
	return l.input[position:l.position]
}

// readLineComment читает однострочный комментарий
func (l *Lexer) readLineComment() string {
	position := l.position
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readBlockComment читает многострочный комментарий
func (l *Lexer) readBlockComment() string {
	position := l.position
	for {
		if l.ch == 0 {
			break
		}
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar()
			l.readChar()
			break
		}
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func (l *Lexer) TokenizeAll() []Token {
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}
	return tokens
}

type Error struct {
	Position Position
	Message  string
}

func (e Error) Error() string {
	return fmt.Sprintf("lexer error at line %d, column %d: %s",
		e.Position.Line, e.Position.Column, e.Message)
}

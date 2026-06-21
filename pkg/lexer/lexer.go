package lexer

type Lexer struct {
	input        string // O código-fonte
	position     int    // Posição atual no input (aponta para o char atual)
	readPosition int    // Próxima posição de leitura (depois do char atual)
	ch           byte   // O caractere atual sendo examinado
	line         int    // Linha atual (para erros)
	column       int    // Coluna atual (para erros) - 1-indexed, reflete l.ch
}

// New cria um novo Lexer.
func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar()
	return l
}

// readChar lê o próximo caractere e avança as posições.
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++

	l.column++
	if l.ch == '\n' {
		l.line++
		l.column = 0
	}
}

// peekChar "espie" o próximo caractere sem avançar.
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// NextToken é a função principal. Lê o input e retorna o próximo token.
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	startLine := l.line
	startColumn := l.column

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_EQUAL_EQUAL, Literal: string(ch) + string(l.ch), Line: startLine, Column: startColumn}
		} else {
			tok = l.newToken(TOKEN_EQUALS, l.ch, startLine, startColumn)
		}
	case ':':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_ASSIGN, Literal: string(ch) + string(l.ch), Line: startLine, Column: startColumn}
		} else {
			tok = l.newToken(TOKEN_COLON, l.ch, startLine, startColumn)
		}
	case ',':
		tok = l.newToken(TOKEN_COMMA, l.ch, startLine, startColumn)
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_GREATER_EQUAL, Literal: ">=", Line: startLine, Column: startColumn}
		} else {
			tok = l.newToken(TOKEN_GREATER, l.ch, startLine, startColumn)
		}
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_LESS_EQUAL, Literal: "<=", Line: startLine, Column: startColumn}
		} else {
			tok = l.newToken(TOKEN_LESS, l.ch, startLine, startColumn)
		}
	case '+':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_PLUS_ASSIGN, Literal: "+=", Line: startLine, Column: startColumn}
		} else {
			tok = l.newToken(TOKEN_PLUS, l.ch, startLine, startColumn)
		}
	case '-':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_MINUS_ASSIGN, Literal: "-=", Line: startLine, Column: startColumn}
		} else {
			tok = l.newToken(TOKEN_MINUS, l.ch, startLine, startColumn)
		}
	case '*':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_ASTERISK_ASSIGN, Literal: "*=", Line: startLine, Column: startColumn}
		} else {
			tok = l.newToken(TOKEN_ASTERISK, l.ch, startLine, startColumn)
		}
	case '/':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_SLASH_ASSIGN, Literal: "/=", Line: startLine, Column: startColumn}
		} else {
			tok = l.newToken(TOKEN_SLASH, l.ch, startLine, startColumn)
		}
	case '%':
		tok = l.newToken(TOKEN_PERCENT, l.ch, startLine, startColumn)
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_NOT_EQUAL, Literal: string(ch) + string(l.ch), Line: startLine, Column: startColumn}
		} else {
			tok = l.newToken(TOKEN_ILLEGAL, l.ch, startLine, startColumn)
		}
	case '(':
		tok = l.newToken(TOKEN_LPAREN, l.ch, startLine, startColumn)
	case ')':
		tok = l.newToken(TOKEN_RPAREN, l.ch, startLine, startColumn)
	case '[':
		tok = l.newToken(TOKEN_LBRACKET, l.ch, startLine, startColumn)
	case ']':
		tok = l.newToken(TOKEN_RBRACKET, l.ch, startLine, startColumn)
	case '"':
		tok.Type = TOKEN_STRING_LIT
		tok.Literal = l.readString()
		tok.Line = startLine
		tok.Column = startColumn
	case 0:
		tok.Literal = ""
		tok.Type = TOKEN_EOF
		tok.Line = startLine
		tok.Column = startColumn
	default:
		if isLetter(l.ch) {
			lit := l.readIdentifier()
			tok.Type = LookupIdent(lit)
			tok.Literal = lit
			tok.Line = startLine
			tok.Column = startColumn
			return tok
		} else if isDigit(l.ch) {
			lit := l.readNumber()
			tok.Type = TOKEN_NUMBER_LIT
			tok.Literal = lit
			tok.Line = startLine
			tok.Column = startColumn
			return tok
		} else {
			tok = l.newToken(TOKEN_ILLEGAL, l.ch, startLine, startColumn)
		}
	}

	l.readChar()
	return tok
}

// newToken cria um token simples de um caractere.
func (l *Lexer) newToken(tokenType TokenType, ch byte, line, col int) Token {
	return Token{Type: tokenType, Literal: string(ch), Line: line, Column: col}
}

// skipWhitespace pula espaços em branco, tabulações, quebras de linha e comentários.
func (l *Lexer) skipWhitespace() {
	for {
		if l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
			l.readChar()
		} else if l.ch == '/' && l.peekChar() == '/' {
			l.skipComment()
		} else if l.ch == '/' && l.peekChar() == '*' {
			l.skipBlockComment()
		} else {
			break
		}
	}
}

// readIdentifier lê um identificador ou palavra-chave.
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber lê um literal numérico (inteiro, flutuante, hex, bin, octal).
func (l *Lexer) readNumber() string {
	position := l.position

	if l.ch == '0' {
		next := l.peekChar()
		if next == 'x' || next == 'X' {
			l.readChar()
			l.readChar()
			for isHexDigit(l.ch) {
				l.readChar()
			}
			return l.input[position:l.position]
		}
		if next == 'b' || next == 'B' {
			l.readChar()
			l.readChar()
			for l.ch == '0' || l.ch == '1' {
				l.readChar()
			}
			return l.input[position:l.position]
		}
		if next == 'o' || next == 'O' {
			l.readChar()
			l.readChar()
			for l.ch >= '0' && l.ch <= '7' {
				l.readChar()
			}
			return l.input[position:l.position]
		}
	}

	for isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}

	if l.ch == '.' {
		l.readChar()
		for isDigit(l.ch) || l.ch == '_' {
			l.readChar()
		}
	}

	if l.ch == 'e' || l.ch == 'E' {
		l.readChar()
		if l.ch == '+' || l.ch == '-' {
			l.readChar()
		}
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[position:l.position]
}

func isHexDigit(ch byte) bool {
	return ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f' || ch >= 'A' && ch <= 'F'
}

// readString lê um literal de string (tudo entre aspas duplas), processando escapes.
func (l *Lexer) readString() string {
	var result []byte
	terminated := false

	for {
		l.readChar()
		if l.ch == '"' {
			terminated = true
			break
		}
		if l.ch == 0 {
			break
		}
		if l.ch == '\\' {
			esc := l.peekChar()
			switch esc {
			case 'n':
				result = append(result, '\n')
				l.readChar()
			case '"':
				result = append(result, '"')
				l.readChar()
			case '\\':
				result = append(result, '\\')
				l.readChar()
			case 't':
				result = append(result, '\t')
				l.readChar()
			case 'r':
				result = append(result, '\r')
				l.readChar()
			default:
				result = append(result, '\\', l.ch)
			}
		} else {
			result = append(result, l.ch)
		}
	}

	if !terminated {
		return "\x00"
	}
	if result == nil {
		return ""
	}
	return string(result)
}

// --- Funções Auxiliares ---

func isLetter(ch byte) bool {
	// Permite letras (a-z, A-Z) e underscore (_) em identificadores
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

func (l *Lexer) skipBlockComment() {
	l.readChar() // *
	l.readChar() // primeiro char depois de /*
	for {
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar()
			l.readChar()
			return
		}
		if l.ch == 0 {
			return
		}
		l.readChar()
	}
}

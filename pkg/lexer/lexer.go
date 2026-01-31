package lexer

type Lexer struct {
	input        string // O código-fonte
	position     int    // Posição atual no input (aponta para o char atual)
	readPosition int    // Próxima posição de leitura (depois do char atual)
	ch           byte   // O caractere atual sendo examinado
	line         int    // Linha atual (para erros)
	column       int    // Coluna atual (para erros)
}

// New cria um novo Lexer.
func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 1}
	l.readChar() // Inicializa o lexer lendo o primeiro char
	return l
}

// readChar lê o próximo caractere e avança as posições.
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		// Chegamos ao fim do arquivo (End of File)
		l.ch = 0 // 0 é o byte nulo (ASCII), sinaliza EOF
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++

	// Atualiza linha/coluna para rastreamento de erro
	if l.ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
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

	// Salva a posição inicial do token
	startLine := l.line
	startColumn := l.column

	switch l.ch {
	case '=':
		// --- LÓGICA ATUALIZADA ---
		// Verifica se é '=='
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: TOKEN_EQUAL_EQUAL, Literal: literal, Line: startLine, Column: startColumn - 1}
		} else {
			tok = l.newToken(TOKEN_EQUALS, l.ch, startLine, startColumn-1)
		}
	case ':':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: TOKEN_ASSIGN, Literal: literal, Line: startLine, Column: startColumn - 1}
		} else {
			tok = l.newToken(TOKEN_COLON, l.ch, startLine, startColumn-1)
		}
	case ',':
		tok = l.newToken(TOKEN_COMMA, l.ch, startLine, startColumn-1)
	case '>':

		tok = l.newToken(TOKEN_GREATER, l.ch, startLine, startColumn-1)
	case '<':

		tok = l.newToken(TOKEN_LESS, l.ch, startLine, startColumn-1)
	case '+':
		tok = l.newToken(TOKEN_PLUS, l.ch, startLine, startColumn-1)
	case '-':
		tok = l.newToken(TOKEN_MINUS, l.ch, startLine, startColumn-1)
	case '*':
		tok = l.newToken(TOKEN_ASTERISK, l.ch, startLine, startColumn-1)
	case '/':
		tok = l.newToken(TOKEN_SLASH, l.ch, startLine, startColumn-1)
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: TOKEN_NOT_EQUAL, Literal: literal, Line: startLine, Column: startColumn - 1}
		} else {
			// (Se não for '!=', '!' sozinho é ilegal)
			tok = l.newToken(TOKEN_ILLEGAL, l.ch, startLine, startColumn-1)
		}
	case '(':
		tok = l.newToken(TOKEN_LPAREN, l.ch, startLine, startColumn-1)
	case ')':
		tok = l.newToken(TOKEN_RPAREN, l.ch, startLine, startColumn-1)
	case '[':
		tok = l.newToken(TOKEN_LBRACKET, l.ch, startLine, startColumn-1)
	case ']':
		tok = l.newToken(TOKEN_RBRACKET, l.ch, startLine, startColumn-1)
	case '"':
		tok.Type = TOKEN_STRING_LIT
		tok.Literal = l.readString()
		tok.Line = startLine
		tok.Column = startColumn - 1 // A coluna da aspa inicial
	case 0:
		tok.Literal = ""
		tok.Type = TOKEN_EOF
		tok.Line = startLine
		tok.Column = startColumn - 1
	default:
		if isLetter(l.ch) {
			// Pode ser um Identificador ou uma Palavra-Chave
			lit := l.readIdentifier()
			tok.Type = LookupIdent(lit) // Verifica se é keyword
			tok.Literal = lit
			tok.Line = startLine
			tok.Column = startColumn - 1
			return tok // readIdentifier já avançou, então retorne
		} else if isDigit(l.ch) {
			// É um número
			lit := l.readNumber()
			tok.Type = TOKEN_NUMBER_LIT
			tok.Literal = lit
			tok.Line = startLine
			tok.Column = startColumn - 1
			return tok // readNumber já avançou, então retorne
		} else {
			// Não sabemos o que é isso
			tok = l.newToken(TOKEN_ILLEGAL, l.ch, startLine, startColumn-1)
		}
	}

	l.readChar() // Avança para o próximo caractere
	return tok
}

// newToken cria um token simples de um caractere.
func (l *Lexer) newToken(tokenType TokenType, ch byte, line, col int) Token {
	return Token{Type: tokenType, Literal: string(ch), Line: line, Column: col}
}

// skipWhitespace pula espaços em branco, tabulações e quebras de linha.
func (l *Lexer) skipWhitespace() {
	for { // Loop para consumir múltiplos espaços ou comentários
		if l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
			// É espaço em branco, consome
			l.readChar()
		} else if l.ch == '/' && l.peekChar() == '/' {
			// É um comentário, pula a linha inteira
			l.skipComment()
		} else {
			// Não é espaço em branco nem comentário, pare
			break
		}
	}
}

// readIdentifier lê um identificador ou palavra-chave.
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) {
		l.readChar()
	}
	// Permite 'nomes_com_underscore' (mas a V1 não usa)
	// for isLetter(l.ch) || l.ch == '_' {
	// 	l.readChar()
	// }
	return l.input[position:l.position]
}

// readNumber lê um literal numérico (apenas inteiros por enquanto).
func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	// TODO: Adicionar suporte a ponto flutuante (ex: 10.5) se V2 precisar
	return l.input[position:l.position]
}

// readString lê um literal de string (tudo entre aspas duplas).
func (l *Lexer) readString() string {
	position := l.position + 1 // Pula o " inicial
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
		// TODO: Adicionar suporte para escapes (ex: \" ou \n)
	}
	// Trata o caso de string não terminada
	if l.ch == 0 {
		// EOF, mas string não fechou. O Parser vai pegar isso.
		return "STRING NÃO TERMINADA" // Provisório
	}

	return l.input[position:l.position]
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
	// O l.ch atual é o primeiro '/', e o peek é o segundo.
	// Continua lendo até encontrar a quebra de linha ou o fim do arquivo.
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

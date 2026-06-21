package lexer

// TokenType é uma string para facilitar a depuração.
type TokenType string

// Token representa uma unidade léxica da linguagem.
type Token struct {
	Type    TokenType // O tipo do token (ex: NÚMERO, STRING)
	Literal string    // O valor literal (ex: "123", "Hello")
	Line    int       // A linha no código-fonte (para erros)
	Column  int       // A coluna no código-fonte (para erros)
}

// Constantes para todos os tipos de tokens
const (
	// Tokens Especiais
	TOKEN_ILLEGAL TokenType = "ILLEGAL" // Token/caractere desconhecido
	TOKEN_EOF     TokenType = "EOF"     // Fim do arquivo (End of File)

	// Identificadores + Literais
	TOKEN_IDENT      TokenType = "IDENT"      // nome, i, versao
	TOKEN_NUMBER_LIT TokenType = "NUMBER_LIT" // 123, 5
	TOKEN_STRING_LIT TokenType = "STRING_LIT" // "Ion", "Olá"

	// --- NOVOS TOKENS V4 (TIPOS) ---
	TOKEN_BOOLEAN_TYPE TokenType = "BOOLEAN_TYPE" // 'boolean'
	TOKEN_TRUE         TokenType = "TRUE"         // 'true'
	TOKEN_FALSE        TokenType = "FALSE"        // 'false'
	TOKEN_NIL          TokenType = "NIL"          // 'nil'

	// --- FIM DOS NOVOS TOKENS V4 ---

	TOKEN_AND TokenType = "AND" // 'and'
	TOKEN_OR  TokenType = "OR"  // 'or'
	TOKEN_NOT TokenType = "NOT" // 'not'

	TOKEN_WHILE    TokenType = "WHILE" // 'while'
	TOKEN_DO       TokenType = "DO"    // 'do'
	TOKEN_FUNCTION TokenType = "FUNCTION"
	TOKEN_RETURN   TokenType = "RETURN"
	TOKEN_BREAK      TokenType = "BREAK"
	TOKEN_CONTINUE   TokenType = "CONTINUE"
	TOKEN_TO_STRING  TokenType = "TO_STRING"
	TOKEN_TO_NUMBER  TokenType = "TO_NUMBER"
	TOKEN_EXIT       TokenType = "EXIT"
	TOKEN_READ_FILE  TokenType = "READ_FILE"
	TOKEN_WRITE_FILE TokenType = "WRITE_FILE"

	// --- NOVOS TOKENS V10 (ARRAYS) ---
	TOKEN_LBRACKET TokenType = "[" // '['
	TOKEN_RBRACKET TokenType = "]" // ']'
	// --- FIM DOS NOVOS TOKENS V10 ---

	// --- NOVOS TOKENS V11 (NATIVAS) ---
	TOKEN_CHAR      TokenType = "CHAR" // 'char'
	TOKEN_ORD       TokenType = "ORD"  // 'ord'
	TOKEN_NOT_EQUAL TokenType = "!="   // '!='
	// --- FIM DOS NOVOS TOKENS V11 ---

	// --- NOVOS TOKENS V6 (INPUT) ---
	TOKEN_INPUT  TokenType = "INPUT" // 'input'
	TOKEN_LPAREN TokenType = "("     // '('
	TOKEN_RPAREN TokenType = ")"     // ')'
	// --- FIM DOS NOVOS TOKENS V6 ---

	TOKEN_LEN         TokenType = "LEN" // 'len'
	TOKEN_GET_BYTE_AT TokenType = "GET_BYTE_AT"
	// Operadores
	TOKEN_ASSIGN TokenType = ":="
	TOKEN_EQUALS TokenType = "="
	TOKEN_COLON  TokenType = ":"
	TOKEN_COMMA  TokenType = ","

	TOKEN_PLUS         TokenType = "+"
	TOKEN_PLUS_ASSIGN  TokenType = "+="
	TOKEN_MINUS        TokenType = "-"
	TOKEN_MINUS_ASSIGN TokenType = "-="
	TOKEN_ASTERISK     TokenType = "*"
	TOKEN_ASTERISK_ASSIGN TokenType = "*="
	TOKEN_SLASH        TokenType = "/"
	TOKEN_SLASH_ASSIGN TokenType = "/="

	TOKEN_EQUAL_EQUAL  TokenType = "==" // Duplo igual
	TOKEN_GREATER      TokenType = ">"  // Maior que
	TOKEN_GREATER_EQUAL TokenType = ">=" // Maior ou igual
	TOKEN_LESS         TokenType = "<"  // Menor que
	TOKEN_LESS_EQUAL   TokenType = "<=" // Menor ou igual
	TOKEN_PERCENT      TokenType = "%"  // Módulo

	// Palavras-chave da Linguagem (V1)
	TOKEN_BEGIN       TokenType = "BEGIN"
	TOKEN_PROGRAM     TokenType = "PROGRAM"
	TOKEN_END         TokenType = "END"
	TOKEN_DECLARE     TokenType = "DECLARE"
	TOKEN_STRING_TYPE TokenType = "STRING_TYPE" // A palavra-chave 'string'
	TOKEN_NUMBER_TYPE TokenType = "NUMBER_TYPE" // A palavra-chave 'number'
	TOKEN_FOR         TokenType = "FOR"
	TOKEN_TO          TokenType = "TO"
	TOKEN_STEP        TokenType = "STEP"
	TOKEN_NEXT        TokenType = "NEXT"
	TOKEN_DISPLAY     TokenType = "DISPLAY"
	TOKEN_IF          TokenType = "IF"
	TOKEN_THEN        TokenType = "THEN"
	TOKEN_ELSE        TokenType = "ELSE"
	TOKEN_ENDIF       TokenType = "ENDIF"
)

// keywords armazena o mapa de palavras-chave da linguagem.
var keywords = map[string]TokenType{
	"begin":       TOKEN_BEGIN,
	"program":     TOKEN_PROGRAM,
	"end":         TOKEN_END,
	"declare":     TOKEN_DECLARE,
	"string":      TOKEN_STRING_TYPE,
	"number":      TOKEN_NUMBER_TYPE,
	"for":         TOKEN_FOR,
	"to":          TOKEN_TO,
	"step":        TOKEN_STEP,
	"next":        TOKEN_NEXT,
	"display":     TOKEN_DISPLAY,
	"if":          TOKEN_IF,
	"then":        TOKEN_THEN,
	"else":        TOKEN_ELSE,
	"endif":       TOKEN_ENDIF,
	"boolean":     TOKEN_BOOLEAN_TYPE,
	"true":        TOKEN_TRUE,
	"false":       TOKEN_FALSE,
	"nil":         TOKEN_NIL,
	"input":       TOKEN_INPUT,
	"and":         TOKEN_AND,
	"or":          TOKEN_OR,
	"not":         TOKEN_NOT,
	"while":       TOKEN_WHILE,
	"do":          TOKEN_DO,
	"function":    TOKEN_FUNCTION,
	"char":        TOKEN_CHAR,
	"ord":         TOKEN_ORD,
	"len":         TOKEN_LEN,
	"get_byte_at": TOKEN_GET_BYTE_AT,
	"return":      TOKEN_RETURN,
	"break":       TOKEN_BREAK,
	"continue":    TOKEN_CONTINUE,
	"tostring":    TOKEN_TO_STRING,
	"tonumber":    TOKEN_TO_NUMBER,
	"exit":        TOKEN_EXIT,
	"readfile":    TOKEN_READ_FILE,
	"writefile":   TOKEN_WRITE_FILE,
}

// LookupIdent verifica se um identificador é uma palavra-chave.
// Se for, retorna o TokenType da palavra-chave.
// Se não, retorna TOKEN_IDENT.
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}

// LookupIdent verifica se um identificador é uma palavra-chave (case-insensitive).
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[toLower(ident)]; ok {
		return tok
	}
	return TOKEN_IDENT
}

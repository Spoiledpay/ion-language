package parser

import (
	"fmt"
	"ion-language/pkg/lexer"
	"strconv"
)

// Definições de precedência (V2)
const (
	_ int = iota
	LOWEST
	LOGICAL_OR  // or
	LOGICAL_AND // and
	EQUALS      // ==
	LESSGREATER // > ou <
	SUM         // + ou -
	PRODUCT     // * ou /
	PREFIX      // not
	CALL        // myFunction()
	INDEX       // array[10]
)

// Mapa de precedências para os operadores V2
var precedences = map[lexer.TokenType]int{
	lexer.TOKEN_EQUAL_EQUAL:     EQUALS,
	lexer.TOKEN_NOT_EQUAL:       EQUALS,
	lexer.TOKEN_LESS:            LESSGREATER,
	lexer.TOKEN_LESS_EQUAL:      LESSGREATER,
	lexer.TOKEN_GREATER:         LESSGREATER,
	lexer.TOKEN_GREATER_EQUAL:   LESSGREATER,
	lexer.TOKEN_PLUS:            SUM,
	lexer.TOKEN_MINUS:           SUM,
	lexer.TOKEN_ASTERISK:        PRODUCT,
	lexer.TOKEN_SLASH:           PRODUCT,
	lexer.TOKEN_PERCENT:         PRODUCT,
	lexer.TOKEN_OR:              LOGICAL_OR,
	lexer.TOKEN_AND:             LOGICAL_AND,
	lexer.TOKEN_LPAREN:          CALL,
	lexer.TOKEN_LBRACKET:        INDEX,
}

type (
	prefixParseFn func() Expression
	infixParseFn  func(Expression) Expression
)

type Parser struct {
	l         *lexer.Lexer
	curToken  lexer.Token
	peekToken lexer.Token
	errors    []string

	// Mapas para o parser Pratt
	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn // HABILITADO
}

// New cria um novo Parser.
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	// --- Funções PREFIX ---
	p.prefixParseFns = make(map[lexer.TokenType]prefixParseFn)
	p.registerPrefix(lexer.TOKEN_IDENT, p.parseIdentifier)
	p.registerPrefix(lexer.TOKEN_NUMBER_LIT, p.parseNumberLiteral)
	p.registerPrefix(lexer.TOKEN_STRING_LIT, p.parseStringLiteral)
	p.registerPrefix(lexer.TOKEN_TRUE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.TOKEN_FALSE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.TOKEN_NIL, p.parseNilLiteral)
	p.registerPrefix(lexer.TOKEN_INPUT, p.parseInputExpression)
	p.registerPrefix(lexer.TOKEN_NOT, p.parsePrefixExpression)
	p.registerPrefix(lexer.TOKEN_LPAREN, p.parseGroupedExpression)

	// --- NOVOS Handlers de PREFIXO V10 (Para Tipos) ---
	p.registerPrefix(lexer.TOKEN_NUMBER_TYPE, p.parseTypeIdentifier)
	p.registerPrefix(lexer.TOKEN_STRING_TYPE, p.parseTypeIdentifier)
	p.registerPrefix(lexer.TOKEN_BOOLEAN_TYPE, p.parseTypeIdentifier)
	p.registerPrefix(lexer.TOKEN_LBRACKET, p.parseArrayTypeNode) // para [number](10)
	p.registerPrefix(lexer.TOKEN_CHAR, p.parseCharExpression)
	p.registerPrefix(lexer.TOKEN_ORD, p.parseOrdExpression)
	p.registerPrefix(lexer.TOKEN_LEN, p.parseLenExpression)
	p.registerPrefix(lexer.TOKEN_GET_BYTE_AT, p.parseGetByteAtExpression)
	p.registerPrefix(lexer.TOKEN_TO_STRING, p.parseToStringExpression)
	p.registerPrefix(lexer.TOKEN_TO_NUMBER, p.parseToNumberExpression)
	p.registerPrefix(lexer.TOKEN_READ_FILE, p.parseReadFileExpression)

	// --- Funções INFIX ---
	p.infixParseFns = make(map[lexer.TokenType]infixParseFn)
	p.registerInfix(lexer.TOKEN_GREATER, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_GREATER_EQUAL, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_LESS, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_LESS_EQUAL, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_EQUAL_EQUAL, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_NOT_EQUAL, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_PLUS, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_MINUS, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_ASTERISK, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_SLASH, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_PERCENT, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_AND, p.parseLogicalExpression)
	p.registerInfix(lexer.TOKEN_OR, p.parseLogicalExpression)
	p.registerInfix(lexer.TOKEN_LPAREN, p.parseCallExpression)

	// --- NOVO Handler de INFIXO V10 ---
	p.registerInfix(lexer.TOKEN_LBRACKET, p.parseIndexExpression) // para tape[10]

	p.nextToken()
	p.nextToken()

	return p
}

// Errors retorna a lista de erros de sintaxe.
func (p *Parser) Errors() []string {
	return p.errors
}

// nextToken avança os tokens.
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) parseGroupedExpression() Expression {
	// p.curToken é '(', nós analisamos o que vem *depois* dele.
	p.nextToken() // Consome o '(' e avança para o início da expressão interna

	// Analisa a expressão interna.
	// Usamos LOWEST para resetar a precedência dentro dos parênteses.
	expr := p.parseExpression(LOWEST)

	// Espera que o próximo token seja ')'
	if !p.expectPeek(lexer.TOKEN_RPAREN) {
		return nil // Faltou ')'
	}

	// Retorna a expressão que estava *dentro* dos parênteses
	return expr
}

// ParseProgram é a função principal que analisa o programa inteiro.
func (p *Parser) ParseProgram() *Program {
	program := &Program{}
	program.Statements = []Statement{}

	// Espera 'begin program'
	if !p.expectCur(lexer.TOKEN_BEGIN) {
		return nil // Erro fatal, não podemos continuar
	}
	if !p.expectPeek(lexer.TOKEN_PROGRAM) {
		return nil // Erro fatal
	}
	p.nextToken() // Avança para o início dos statements

	// Analisa todas as instruções até 'end'
	for !p.curTokenIs(lexer.TOKEN_END) && !p.curTokenIs(lexer.TOKEN_EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	// Espera 'end program'
	if !p.curTokenIs(lexer.TOKEN_END) {
		p.newError("esperava 'end' no final do programa, mas recebeu %s", p.curToken.Type)
		return program
	}
	if !p.expectPeek(lexer.TOKEN_PROGRAM) {
		// Retorna o que temos, mas com o erro registrado
		return program
	}

	return program
}

// --- Funções de Roteamento e Registro ---

func (p *Parser) registerPrefix(tokenType lexer.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

// NOVA FUNÇÃO DE REGISTRO
func (p *Parser) registerInfix(tokenType lexer.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

// --- Funções de Análise (Statements) ---

// parseStatement é o roteador principal para instruções.
// parseStatement (Corrigido para V10.1)
func (p *Parser) parseStatement() Statement {
	switch p.curToken.Type {
	case lexer.TOKEN_DECLARE:
		return p.parseDeclareStatement()
	case lexer.TOKEN_DISPLAY:
		return p.parseDisplayStatement()
	case lexer.TOKEN_FOR:
		return p.parseForStatement()
	case lexer.TOKEN_IF:
		return p.parseIfStatement()
	case lexer.TOKEN_WHILE:
		return p.parseWhileStatement()
	case lexer.TOKEN_FUNCTION:
		return p.parseFunctionStatement()
	case lexer.TOKEN_RETURN:
		return p.parseReturnStatement()
	case lexer.TOKEN_BREAK:
		return &BreakStatement{Token: p.curToken}
	case lexer.TOKEN_CONTINUE:
		return &ContinueStatement{Token: p.curToken}
	case lexer.TOKEN_EXIT:
		return p.parseExitStatement()
	case lexer.TOKEN_WRITE_FILE:
		return p.parseWriteFileStatement()

	default:
		// Esta é a nova lógica V10.
		return p.parseAssignmentOrExpressionStatement()
	}
}

func (p *Parser) parseReturnStatement() *ReturnStatement {
	stmt := &ReturnStatement{Token: p.curToken}

	p.nextToken() // Consome o 'return'

	// Analisa a expressão do valor de retorno
	stmt.ReturnValue = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseCharExpression() Expression {
	expr := &CharExpression{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_LPAREN) { // '('
		return nil
	}
	p.nextToken() // Avança para o início da expressão do argumento

	expr.Argument = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_RPAREN) { // ')'
		return nil
	}

	return expr
}

// parseOrdExpression analisa: ord(expr)
func (p *Parser) parseOrdExpression() Expression {
	expr := &OrdExpression{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_LPAREN) {
		return nil
	}
	p.nextToken()
	expr.Argument = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_RPAREN) {
		return nil
	}

	return expr
}

func (p *Parser) parseWhileStatement() *WhileStatement {
	stmt := &WhileStatement{Token: p.curToken}

	p.nextToken() // Consome o 'while'

	// Analisa a condição
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_DO) {
		return nil
	}
	p.nextToken() // Consome o 'do'

	// Analisa o corpo (Body)
	stmt.Body = []Statement{}

	// Continua analisando statements até encontrar o 'end'
	for !p.curTokenIs(lexer.TOKEN_END) && !p.curTokenIs(lexer.TOKEN_EOF) {
		s := p.parseStatement()
		if s != nil {
			stmt.Body = append(stmt.Body, s)
		}
		p.nextToken()
	}

	// Espera 'end do'
	if !p.curTokenIs(lexer.TOKEN_END) {
		p.newError("bloco 'while' não foi fechado com 'end'. Recebeu: %s", p.curToken.Type)
		return stmt // Retorna o que tem, mesmo com erro
	}

	if !p.expectPeek(lexer.TOKEN_DO) {
		p.newError("esperava 'do' após 'end' para fechar o bloco 'while'.")
		return nil
	}

	return stmt
}

/*
func (p *Parser) parseAssignmentStatement() *AssignmentStatement {
	// p.curToken é o IDENTIFICADOR
	stmt := &AssignmentStatement{
		Token: p.curToken,
		Name:  &Identifier{Token: p.curToken, Value: p.curToken.Literal},
	}

	// Consome o IDENT (p.curToken) e o ASSIGN (p.peekToken)
	p.nextToken() // Avança p.curToken para :=
	p.nextToken() // Avança p.curToken para o início da expressão

	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}
*/

// parseDeclareStatement (Não muda)
// parseDeclareStatement (Corrigida para V10)
func (p *Parser) parseDeclareStatement() *DeclareStatement {
	// A struct DeclareStatement em 'ast.go' deve ter 'TypeNode Expression'
	stmt := &DeclareStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}
	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_COLON) {
		return nil
	}
	p.nextToken() // Avança para o início do TIPO

	// --- ESTA É A MUDANÇA V10 ---
	// Em vez de verificar 'string' ou 'number',
	// chamamos 'parseType()' (que sabe analisar 'number' E '[number](10)')
	stmt.TypeNode = p.parseType()
	// --- FIM DA MUDANÇA V10 ---

	if stmt.TypeNode == nil {
		p.newError("esperava um tipo (number, [string](10), etc) após ':'")
		return nil
	}

	// Verifica se há uma inicialização [ := <expression> ]
	if p.peekTokenIs(lexer.TOKEN_ASSIGN) {
		p.nextToken() // Avança para :=
		p.nextToken() // Avança para o início da expressão
		stmt.Value = p.parseExpression(LOWEST)
	}

	return stmt
}

// parseDisplayStatement (Não muda)
func (p *Parser) parseDisplayStatement() *DisplayStatement {
	stmt := &DisplayStatement{Token: p.curToken}
	stmt.Arguments = []Expression{}

	p.nextToken() // Avança do 'display' para a primeira expressão
	stmt.Arguments = append(stmt.Arguments, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.TOKEN_COMMA) {
		p.nextToken() // Avança para a VÍRGULA
		p.nextToken() // Avança para a próxima expressão
		stmt.Arguments = append(stmt.Arguments, p.parseExpression(LOWEST))
	}

	return stmt
}

// parseForStatement (Não muda)
func (p *Parser) parseForStatement() *ForStatement {
	stmt := &ForStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}
	stmt.Counter = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_EQUALS) {
		return nil
	}
	p.nextToken() // Avança para a expressão 'start'
	stmt.Start = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_TO) {
		return nil
	}
	p.nextToken() // Avança para a expressão 'end'
	stmt.End = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.TOKEN_STEP) {
		p.nextToken()
		p.nextToken()
		stmt.Step = p.parseExpression(LOWEST)
	} else {
		stmt.Step = &NumberLiteral{Token: lexer.Token{Line: p.curToken.Line}, Value: 1}
	}

	// Analisa o corpo do loop
	stmt.Body = []Statement{}
	p.nextToken() // Avança para o início do corpo

	for !p.curTokenIs(lexer.TOKEN_NEXT) && !p.curTokenIs(lexer.TOKEN_EOF) {
		s := p.parseStatement()
		if s != nil {
			stmt.Body = append(stmt.Body, s)
		}
		p.nextToken()
	}

	if !p.curTokenIs(lexer.TOKEN_NEXT) {
		p.newError("bloco 'for' não foi fechado com 'next'. Recebeu: %s", p.curToken.Type)
		return stmt
	}

	return stmt
}

// NOVA FUNÇÃO V2
// parseIfStatement analisa: if <condition> then <consequence> [else <alternative>] endif
func (p *Parser) parseIfStatement() *IfStatement {
	stmt := &IfStatement{Token: p.curToken}

	p.nextToken() // Consome o 'if'
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_THEN) {
		return nil
	}
	p.nextToken() // Consome o 'then'

	// Analisa o bloco 'then'
	stmt.Consequence = p.parseBlockStatement()

	// Verifica se há um bloco 'else'
	if p.curTokenIs(lexer.TOKEN_ELSE) {
		p.nextToken() // Consome o 'else'
		stmt.Alternative = p.parseBlockStatement()
	}

	// O parseBlockStatement() para no 'endif',
	// então o curToken deve ser 'endif' aqui.
	if !p.curTokenIs(lexer.TOKEN_ENDIF) {
		p.newError("bloco 'if' não foi fechado com 'endif'. Recebeu: %s", p.curToken.Type)
		return nil
	}

	return stmt
}

func (p *Parser) parseAssignmentOrExpressionStatement() Statement {
	// --- CORREÇÃO V10.1 ---
	// Precisamos salvar o token *antes* de 'parseExpression' o consumir
	startToken := p.curToken
	// --- FIM DA CORREÇÃO ---

	// 1. Analisa o lado esquerdo como uma expressão completa
	leftExpr := p.parseExpression(LOWEST)

	// 2. Verifica se o próximo token é operador de atribuição
	assignTokens := map[lexer.TokenType]string{
		lexer.TOKEN_ASSIGN:          ":=",
		lexer.TOKEN_PLUS_ASSIGN:     "+=",
		lexer.TOKEN_MINUS_ASSIGN:    "-=",
		lexer.TOKEN_ASTERISK_ASSIGN: "*=",
		lexer.TOKEN_SLASH_ASSIGN:    "/=",
	}
	if op, ok := assignTokens[p.peekToken.Type]; ok {
		p.nextToken()
		stmt := &AssignmentStatement{
			Token:    p.curToken,
			Left:     leftExpr,
			Operator: op,
		}
		p.nextToken()
		stmt.Value = p.parseExpression(LOWEST)
		return stmt
	}

	// 3. Se não for atribuição, é um 'ExpressionStatement' (ex: uma chamada de função)
	stmt := &ExpressionStatement{
		// --- CORREÇÃO V10.1 ---
		Token: startToken, // Usa o token que salvamos
		// --- FIM DA CORREÇÃO ---
		Expression: leftExpr,
	}
	return stmt
}

// parseTypeIdentifier (NOVO V10)
func (p *Parser) parseTypeIdentifier() Expression {
	return &TypeIdentifier{Token: p.curToken}
}

// parseArrayTypeNode (NOVO V10)
func (p *Parser) parseArrayTypeNode() Expression {
	node := &ArrayTypeNode{Token: p.curToken} // O token '['

	// Analisa o tipo base (number, string, etc.)
	if !p.peekTokenIs(lexer.TOKEN_NUMBER_TYPE) &&
		!p.peekTokenIs(lexer.TOKEN_STRING_TYPE) &&
		!p.peekTokenIs(lexer.TOKEN_BOOLEAN_TYPE) {
		p.newError("esperava um tipo (number, string, boolean) dentro de []")
		return nil
	}
	p.nextToken()
	node.BaseType = p.curToken

	if !p.expectPeek(lexer.TOKEN_RBRACKET) { // ']'
		return nil
	}

	// Analisa o tamanho ( <tamanho> )
	if !p.expectPeek(lexer.TOKEN_LPAREN) { // '('
		p.newError("esperava '(' (tamanho) após [tipo]")
		return nil
	}
	p.nextToken() // Consome '('
	node.Size = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_RPAREN) { // ')'
		return nil
	}

	return node
}

// parseType (NOVO V10)
func (p *Parser) parseType() Expression {
	// A função parseExpression fará isso por nós
	// com os 'prefix' handlers que registramos
	return p.parseExpression(LOWEST)
}

// parseIndexExpression (NOVO V10)
func (p *Parser) parseIndexExpression(left Expression) Expression {
	expr := &IndexExpression{
		Token: p.curToken, // O token '['
		Left:  left,
	}
	p.nextToken() // Consome '['
	expr.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_RBRACKET) { // ']'
		return nil
	}

	return expr
}

// parseBlockStatement analisa um bloco de código até encontrar um 'else' ou 'endif'
func (p *Parser) parseBlockStatement() []Statement {
	stmts := []Statement{}

	// V10: Adicionado !p.curTokenIs(lexer.TOKEN_END)
	for !p.curTokenIs(lexer.TOKEN_ELSE) && !p.curTokenIs(lexer.TOKEN_ENDIF) && !p.curTokenIs(lexer.TOKEN_END) && !p.curTokenIs(lexer.TOKEN_EOF) {
		s := p.parseStatement()
		if s != nil {
			stmts = append(stmts, s)
		}
		p.nextToken()
	}

	return stmts
}

// --- Funções de Análise (Expressões) ---

// FUNÇÃO ATUALIZADA (CRÍTICO)
// parseExpression é o ponto de entrada para analisar uma expressão (Pratt Parser).
func (p *Parser) parseExpression(precedence int) Expression {
	// 1. Análise Prefix (ex: 5, "hello", i)
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.newError("nenhuma função de análise (prefix) encontrada para %s", p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	// 2. Loop de Análise Infix (ex: x > 5)
	for precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			// Se o próximo token não for um operador infix
			// que conhecemos, terminamos a expressão.
			return leftExp
		}

		p.nextToken() // Avança para o token do operador (ex: >)

		leftExp = infix(leftExp) // Chama parseInfixExpression
	}

	return leftExp
}

// NOVA FUNÇÃO V2
// parseInfixExpression é chamada quando encontramos um operador infix (>, <, ==)
func (p *Parser) parseInfixExpression(left Expression) Expression {
	// Cria o nó da AST
	expression := &InfixExpression{
		Token: p.curToken, // O token do operador (ex: >)
		Left:  left,       // A expressão que veio da esquerda
	}

	// Obtém a precedência do operador atual
	precedence := p.curPrecedence()

	p.nextToken() // Consome o token do operador

	// Analisa a expressão da direita, com a precedência do operador atual
	expression.Right = p.parseExpression(precedence)

	return expression
}

// --- Funções de Análise Prefix (Não mudam) ---

func (p *Parser) parseIdentifier() Expression {
	return &Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseStringLiteral() Expression {
	return &StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseNumberLiteral() Expression {
	lit := &NumberLiteral{Token: p.curToken}

	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.newError("não foi possível converter '%s' para número: %s", p.curToken.Literal, err)
		return nil
	}
	lit.Value = value
	return lit
}

// --- Funções Auxiliares de Verificação e Erro ---

// NOVAS FUNÇÕES HELPER DE PRECEDÊNCIA
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

// --- (Resto das funções auxiliares: curTokenIs, peekTokenIs, expectCur, expectPeek, peekError, newError) ---
// --- (Elas não mudam) ---

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectCur(t lexer.TokenType) bool {
	if p.curTokenIs(t) {
		return true
	}
	p.newError("token atual inválido. Esperava %s, recebeu %s (Linha: %d, Col: %d)",
		t, p.curToken.Type, p.curToken.Line, p.curToken.Column)
	return false
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t lexer.TokenType) {
	p.newError("esperava o próximo token como %s, mas recebeu %s (Linha: %d, Col: %d)",
		t, p.peekToken.Type, p.peekToken.Line, p.peekToken.Column)
}

func (p *Parser) newError(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	p.errors = append(p.errors, msg)
}

func (p *Parser) parseBooleanLiteral() Expression {
	return &BooleanLiteral{
		Token: p.curToken,
		Value: p.curTokenIs(lexer.TOKEN_TRUE), // true se o token for TOKEN_TRUE
	}
}

func (p *Parser) parseNilLiteral() Expression {
	return &NilLiteral{Token: p.curToken}
}

func (p *Parser) parseInputExpression() Expression {
	expr := &InputExpression{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_LPAREN) {
		// Erro: esperava '(' após 'input'
		return nil
	}

	p.nextToken() // Avança para o início da expressão do prompt

	// Analisa a expressão do prompt
	expr.Prompt = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_RPAREN) {
		// Erro: esperava ')' após a expressão
		return nil
	}

	return expr
}

func (p *Parser) parsePrefixExpression() Expression {
	expr := &PrefixExpression{
		Token: p.curToken, // O token 'not'
	}

	p.nextToken() // Consome o 'not'

	// Analisa a expressão à direita, usando a precedência PREFIX
	expr.Right = p.parseExpression(PREFIX)

	return expr
}

// parseLogicalExpression analisa: <esquerda> and <direita>  OU  <esquerda> or <direita>
func (p *Parser) parseLogicalExpression(left Expression) Expression {
	expr := &LogicalExpression{
		Token:    p.curToken, // O token 'and' ou 'or'
		Left:     left,
		Operator: p.curToken.Literal,
	}

	precedence := p.curPrecedence()
	p.nextToken() // Consome o 'and' ou 'or'
	expr.Right = p.parseExpression(precedence)

	return expr
}

func (p *Parser) parseExpressionStatement() *ExpressionStatement {
	// Reutiliza o ExpressionStatement que definimos na V2
	stmt := &ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.parseExpression(LOWEST)

	return stmt
}

// parseCallExpression analisa: <ident>(<args...>)
func (p *Parser) parseCallExpression(function Expression) Expression {
	// 'function' é a expressão da esquerda (o Identifier)
	// p.curToken é '('
	expr := &CallExpression{
		Token:    p.curToken, // O token '('
		Function: function,   // O Identifier (ex: "Saudacao")
	}

	expr.Arguments = p.parseCallArguments() // Helper para ler os argumentos
	return expr
}

// parseCallArguments analisa os argumentos dentro de '()'
func (p *Parser) parseCallArguments() []Expression {
	args := []Expression{}

	// Caso 1: Chamada vazia, ex: Saudacao()
	if p.peekTokenIs(lexer.TOKEN_RPAREN) {
		p.nextToken() // Consome ')'
		return args
	}

	// Caso 2: Pelo menos um argumento
	p.nextToken() // Avança para a primeira expressão de argumento
	args = append(args, p.parseExpression(LOWEST))

	// Loop para argumentos adicionais
	for p.peekTokenIs(lexer.TOKEN_COMMA) {
		p.nextToken() // Consome ','
		p.nextToken() // Avança para a próxima expressão
		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(lexer.TOKEN_RPAREN) {
		return nil // Faltou ')'
	}

	return args
}

// parseFunctionStatement analisa: function <nome>(<params...>) <body> end function
func (p *Parser) parseFunctionStatement() *FunctionStatement {
	stmt := &FunctionStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) { // Nome da função
		return nil
	}
	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_LPAREN) { // '('
		return nil
	}
	// curToken é '('

	stmt.Parameters = p.parseFunctionParameters()

	p.nextToken() // Avança para o início do corpo

	// Analisa o corpo
	stmt.Body = []Statement{}
	for !p.curTokenIs(lexer.TOKEN_END) && !p.curTokenIs(lexer.TOKEN_EOF) {
		s := p.parseStatement()
		if s != nil {
			stmt.Body = append(stmt.Body, s)
		}
		p.nextToken()
	}

	if !p.curTokenIs(lexer.TOKEN_END) {
		p.newError("bloco 'function' não foi fechado com 'end'. Recebeu: %s", p.curToken.Type)
		return stmt
	}

	if !p.expectPeek(lexer.TOKEN_FUNCTION) {
		p.newError("esperava 'function' após 'end' para fechar o bloco.")
		return nil
	}

	return stmt
}

func (p *Parser) parseFunctionParameters() []Parameter {
	params := []Parameter{}

	if p.peekTokenIs(lexer.TOKEN_RPAREN) {
		p.nextToken()
		return params
	}

	p.nextToken()
	param := Parameter{
		Name: &Identifier{Token: p.curToken, Value: p.curToken.Literal},
	}
	if p.peekTokenIs(lexer.TOKEN_COLON) {
		p.nextToken()
		p.nextToken()
		param.Type = p.parseType()
	}
	params = append(params, param)

	for p.peekTokenIs(lexer.TOKEN_COMMA) {
		p.nextToken()
		p.nextToken()
		param := Parameter{
			Name: &Identifier{Token: p.curToken, Value: p.curToken.Literal},
		}
		if p.peekTokenIs(lexer.TOKEN_COLON) {
			p.nextToken()
			p.nextToken()
			param.Type = p.parseType()
		}
		params = append(params, param)
	}

	if !p.expectPeek(lexer.TOKEN_RPAREN) {
		return nil
	}

	return params
}

// ... (Depois de parseOrdExpression)

// --- NOVAS FUNÇÕES DE PARSING V12.B ---

// parseLenExpression analisa: len( <expressão> )
func (p *Parser) parseLenExpression() Expression {
	expr := &LenExpression{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_LPAREN) { // '('
		return nil
	}
	p.nextToken() // Avança para o início da expressão do argumento

	expr.Argument = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_RPAREN) { // ')'
		return nil
	}

	return expr
}

// parseGetByteAtExpression analisa: get_byte_at( <string>, <index> )
func (p *Parser) parseGetByteAtExpression() Expression {
	expr := &GetByteAtExpression{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_LPAREN) { // '('
		return nil
	}

	// Analisa o Argumento 1 (Target)
	p.nextToken()
	expr.Target = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_COMMA) { // ','
		p.newError("esperava ',' (vírgula) após o primeiro argumento de get_byte_at")
		return nil
	}

	// Analisa o Argumento 2 (Index)
	p.nextToken()
	expr.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_RPAREN) { // ')'
		return nil
	}

	return expr
}

func (p *Parser) parseToStringExpression() Expression {
	expr := &ToStringExpression{Token: p.curToken}
	if !p.expectPeek(lexer.TOKEN_LPAREN) { return nil }
	p.nextToken()
	expr.Argument = p.parseExpression(LOWEST)
	if !p.expectPeek(lexer.TOKEN_RPAREN) { return nil }
	return expr
}

func (p *Parser) parseToNumberExpression() Expression {
	expr := &ToNumberExpression{Token: p.curToken}
	if !p.expectPeek(lexer.TOKEN_LPAREN) { return nil }
	p.nextToken()
	expr.Argument = p.parseExpression(LOWEST)
	if !p.expectPeek(lexer.TOKEN_RPAREN) { return nil }
	return expr
}

func (p *Parser) parseReadFileExpression() Expression {
	expr := &ReadFileExpression{Token: p.curToken}
	if !p.expectPeek(lexer.TOKEN_LPAREN) { return nil }
	p.nextToken()
	expr.Path = p.parseExpression(LOWEST)
	if !p.expectPeek(lexer.TOKEN_RPAREN) { return nil }
	return expr
}

func (p *Parser) parseExitStatement() *ExitStatement {
	stmt := &ExitStatement{Token: p.curToken}
	if !p.expectPeek(lexer.TOKEN_LPAREN) { return nil }
	p.nextToken()
	stmt.Code = p.parseExpression(LOWEST)
	if !p.expectPeek(lexer.TOKEN_RPAREN) { return nil }
	return stmt
}

func (p *Parser) parseWriteFileStatement() *WriteFileStatement {
	stmt := &WriteFileStatement{Token: p.curToken}
	if !p.expectPeek(lexer.TOKEN_LPAREN) { return nil }
	p.nextToken()
	stmt.Path = p.parseExpression(LOWEST)
	if !p.expectPeek(lexer.TOKEN_COMMA) {
		p.newError("esperava ',' após o caminho em writeFile")
		return nil
	}
	p.nextToken()
	stmt.Content = p.parseExpression(LOWEST)
	if !p.expectPeek(lexer.TOKEN_RPAREN) { return nil }
	return stmt
}

// --- Funções Auxiliares de Verificação e Erro ---

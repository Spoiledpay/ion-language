package parser

import "ion-language/pkg/lexer"

// Todos os nós da AST devem implementar esta interface.
// Node é a interface base.
type Node interface {
	// TokenLiteral retorna o literal do token associado ao nó.
	// Usado principalmente para depuração.
	TokenLiteral() string
}

// Statement (Declaração/Instrução) é um tipo de nó que não produz um valor.
// Ex: declare, for, display
type Statement interface {
	Node
	statementNode() // Método "fantasma" para garantir tipo
}

// Expression (Expressão) é um tipo de nó que produz um valor.
// Ex: 5, "Hello", i, 1 + 2
type Expression interface {
	Node
	expressionNode() // Método "fantasma" para garantir tipo
}

// --- Nós Principais ---

// Program é o nó raiz da AST.
// Todo programa Ion é uma sequência de Statements.
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

// --- Nós de Expressão ---

// Identifier representa um nome de variável.
// Implementa a interface 'Expression'.
type Identifier struct {
	Token lexer.Token // O token TOKEN_IDENT
	Value string      // O nome da variável, ex: "i", "nome"
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }

// NumberLiteral representa um literal numérico.
// Implementa a interface 'Expression'.
type NumberLiteral struct {
	Token lexer.Token // O token TOKEN_NUMBER_LIT
	Value float64     // Armazenamos como float para futura expansão
}

func (nl *NumberLiteral) expressionNode()      {}
func (nl *NumberLiteral) TokenLiteral() string { return nl.Token.Literal }

// StringLiteral representa um literal de string.
// Implementa a interface 'Expression'.
type StringLiteral struct {
	Token lexer.Token // O token TOKEN_STRING_LIT
	Value string      // O valor da string, ex: "Ion"
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }

// --- Nós de Instrução (Statements) ---

// DeclareStatement representa a instrução 'declare'
// Ex: declare i: number := 10
type DeclareStatement struct {
	Token    lexer.Token // O token TOKEN_DECLARE
	Name     *Identifier // O 'i'
	TypeNode Expression  // <--- MUDANÇA: 'Type' virou 'TypeNode' e é uma Expression
	Value    Expression  // O '10' (pode ser nulo)
}

func (ds *DeclareStatement) statementNode()       {}
func (ds *DeclareStatement) TokenLiteral() string { return ds.Token.Literal }

// DisplayStatement representa a instrução 'display'
// Ex: display "Valor: ", i
type DisplayStatement struct {
	Token lexer.Token // O token TOKEN_DISPLAY
	// A instrução 'display' pode ter múltiplos argumentos separados por vírgula
	Arguments []Expression
}

func (ds *DisplayStatement) statementNode()       {}
func (ds *DisplayStatement) TokenLiteral() string { return ds.Token.Literal }

// ForStatement representa o loop 'for ... next'
// Ex: for i = 1 to 5 step 1 ... next
type ForStatement struct {
	Token   lexer.Token // O token TOKEN_FOR
	Counter *Identifier // O 'i'
	Start   Expression  // O '1'
	End     Expression  // O '5'
	Step    Expression  // O '1'
	Body    []Statement // O que está dentro do loop
}

func (fs *ForStatement) statementNode()       {}
func (fs *ForStatement) TokenLiteral() string { return fs.Token.Literal }

// ExpressionStatement é um "wrapper"
// Em Ion V1, não temos isso, mas linguagens como Go/C têm (ex: i++).
// Vamos deixar aqui para referência futura.
// Por enquanto, não vamos usá-lo no parser.
type ExpressionStatement struct {
	Token      lexer.Token // O primeiro token da expressão
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }

type InfixExpression struct {
	Token lexer.Token // O token do operador, ex: >
	Left  Expression  // A expressão da esquerda (ex: x)
	Right Expression  // A expressão da direita (ex: 5)
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }

type IfStatement struct {
	Token       lexer.Token // O token 'if'
	Condition   Expression  // A condição (ex: InfixExpression)
	Consequence []Statement // O bloco 'then'
	Alternative []Statement // O bloco 'else' (pode ser nulo)
}

func (is *IfStatement) statementNode()       {}
func (is *IfStatement) TokenLiteral() string { return is.Token.Literal }

// ... (Depois da struct IfStatement)

// --- NOVOS NÓS DA AST (V4) ---

// BooleanLiteral representa os literais 'true' e 'false'
// Implementa a interface 'Expression'.
type BooleanLiteral struct {
	Token lexer.Token // O token TOKEN_TRUE ou TOKEN_FALSE
	Value bool
}

func (bl *BooleanLiteral) expressionNode()      {}
func (bl *BooleanLiteral) TokenLiteral() string { return bl.Token.Literal }

// NilLiteral representa o literal 'nil'
// Implementa a interface 'Expression'.
type NilLiteral struct {
	Token lexer.Token // O token TOKEN_NIL
}

func (nl *NilLiteral) expressionNode()      {}
func (nl *NilLiteral) TokenLiteral() string { return nl.Token.Literal }

// --- NOVO NÓ DA AST (V5) ---

// AssignmentStatement representa uma re-atribuição de variável
// Ex: i := 10, i += 5
type AssignmentStatement struct {
	Token    lexer.Token // O token := (ASSIGN) ou +=, -=, *=, /=
	Left     Expression  // O que está sendo atribuído (Identifier ou IndexExpr)
	Operator string      // ":=", "+=", "-=", "*=", "/="
	Value    Expression  // O valor
}

func (as *AssignmentStatement) statementNode()       {}
func (as *AssignmentStatement) TokenLiteral() string { return as.Token.Literal }

// --- NOVO NÓ DA AST (V6) ---

// InputExpression representa a chamada da função 'input("prompt")'
// Implementa a interface 'Expression'.
type InputExpression struct {
	Token  lexer.Token // O token TOKEN_INPUT
	Prompt Expression  // A expressão do prompt (ex: "Qual o seu nome?")
}

func (ie *InputExpression) expressionNode()      {}
func (ie *InputExpression) TokenLiteral() string { return ie.Token.Literal }

type PrefixExpression struct {
	Token lexer.Token // O token do prefixo, ex: TOKEN_NOT
	Right Expression  // A expressão à direita do token
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }

// LogicalExpression representa uma operação lógica infixa
// Ex: a and b, a or b
// Implementa a interface 'Expression'.
type LogicalExpression struct {
	Token    lexer.Token // O token do operador (TOKEN_AND ou TOKEN_OR)
	Left     Expression  // A expressão da esquerda
	Operator string      // "and" ou "or" (para o compilador)
	Right    Expression  // A expressão da direita
}

func (le *LogicalExpression) expressionNode()      {}
func (le *LogicalExpression) TokenLiteral() string { return le.Token.Literal }

// WhileStatement representa um loop 'while...do...end do'
// Implementa a interface 'Statement'.
type WhileStatement struct {
	Token     lexer.Token // O token TOKEN_WHILE
	Condition Expression  // A condição (ex: x < 5)
	Body      []Statement // O bloco 'do'
}

func (ws *WhileStatement) statementNode()       {}
func (ws *WhileStatement) TokenLiteral() string { return ws.Token.Literal }

type Parameter struct {
	Name *Identifier
	Type Expression // pode ser nil se sem tipo
}

type FunctionStatement struct {
	Token      lexer.Token
	Name       *Identifier
	Parameters []Parameter
	Body       []Statement
}

func (fs *FunctionStatement) statementNode()       {}
func (fs *FunctionStatement) TokenLiteral() string { return fs.Token.Literal }

// CallExpression representa a chamada de uma função
// Ex: Saudacao("Ion")
// Implementa a interface 'Expression'.
type CallExpression struct {
	Token     lexer.Token  // O token '('
	Function  Expression   // O nome da função (um Identifier)
	Arguments []Expression // Os argumentos (ex: "Ion")
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }

type ArrayTypeLiteral struct {
	Token    lexer.Token // O token '[' (LBRACKET)
	BaseType lexer.Token // O token do tipo (ex: TOKEN_NUMBER_TYPE)
}

func (atl *ArrayTypeLiteral) expressionNode()      {}
func (atl *ArrayTypeLiteral) TokenLiteral() string { return atl.Token.Literal }

// IndexExpression representa um acesso de índice
// Ex: tape[10]
// Implementa a interface 'Expression'.
type IndexExpression struct {
	Token lexer.Token // O token '[' (LBRACKET)
	Left  Expression  // A expressão da esquerda (o array, ex: "tape")
	Index Expression  // A expressão do índice (ex: 10)
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }

type TypeIdentifier struct {
	Token lexer.Token // O token do tipo (ex: TOKEN_NUMBER_TYPE)
}

func (ti *TypeIdentifier) expressionNode()      {}
func (ti *TypeIdentifier) TokenLiteral() string { return ti.Token.Literal }

type ArrayTypeNode struct {
	Token    lexer.Token // O token '[' (LBRACKET)
	BaseType lexer.Token // O token do tipo (ex: TOKEN_NUMBER_TYPE)
	Size     Expression  // A expressão do tamanho (ex: 30000)
}

func (atn *ArrayTypeNode) expressionNode()      {}
func (atn *ArrayTypeNode) TokenLiteral() string { return atn.Token.Literal }

// ... (Depois de IndexExpression)

// --- NOVOS NÓS DA AST (V11) ---

// CharExpression representa a chamada da função nativa 'char(number)'
// Implementa a interface 'Expression'.
type CharExpression struct {
	Token    lexer.Token // O token TOKEN_CHAR
	Argument Expression  // A expressão do argumento (ex: 65)
}

func (ce *CharExpression) expressionNode()      {}
func (ce *CharExpression) TokenLiteral() string { return ce.Token.Literal }

// OrdExpression representa a chamada da função nativa 'ord(expr)'
type OrdExpression struct {
	Token    lexer.Token
	Argument Expression
}

func (oe *OrdExpression) expressionNode()      {}
func (oe *OrdExpression) TokenLiteral() string { return oe.Token.Literal }

// BreakStatement representa a instrução 'break'
type BreakStatement struct {
	Token lexer.Token
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }

// ContinueStatement representa a instrução 'continue'
type ContinueStatement struct {
	Token lexer.Token
}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }

// ... (Depois de OrdExpression)

// --- NOVOS NÓS DA AST (V12.B) ---

// LenExpression representa a chamada da função nativa 'len(string)'
// Implementa a interface 'Expression'.
type LenExpression struct {
	Token    lexer.Token // O token TOKEN_LEN
	Argument Expression  // A expressão do argumento (a string)
}

func (le *LenExpression) expressionNode()      {}
func (le *LenExpression) TokenLiteral() string { return le.Token.Literal }

// GetByteAtExpression representa a chamada da função nativa 'get_byte_at(string, index)'
// Implementa a interface 'Expression'.
type GetByteAtExpression struct {
	Token  lexer.Token // O token TOKEN_GET_BYTE_AT
	Target Expression  // A string
	Index  Expression  // O índice
}

func (gbe *GetByteAtExpression) expressionNode()      {}
func (gbe *GetByteAtExpression) TokenLiteral() string { return gbe.Token.Literal }

// ToStringExpression representa toString(expr)
type ToStringExpression struct {
	Token    lexer.Token
	Argument Expression
}

func (tse *ToStringExpression) expressionNode()      {}
func (tse *ToStringExpression) TokenLiteral() string { return tse.Token.Literal }

// ToNumberExpression representa toNumber(expr)
type ToNumberExpression struct {
	Token    lexer.Token
	Argument Expression
}

func (tne *ToNumberExpression) expressionNode()      {}
func (tne *ToNumberExpression) TokenLiteral() string { return tne.Token.Literal }

// ExitStatement representa exit(code)
type ExitStatement struct {
	Token    lexer.Token
	Code Expression
}

func (es *ExitStatement) statementNode()       {}
func (es *ExitStatement) TokenLiteral() string { return es.Token.Literal }

// ReadFileExpression representa readFile(path)
type ReadFileExpression struct {
	Token lexer.Token
	Path  Expression
}

func (rfe *ReadFileExpression) expressionNode()      {}
func (rfe *ReadFileExpression) TokenLiteral() string { return rfe.Token.Literal }

// WriteFileStatement representa writeFile(path, content)
type WriteFileStatement struct {
	Token   lexer.Token
	Path    Expression
	Content Expression
}

func (wfs *WriteFileStatement) statementNode()       {}
func (wfs *WriteFileStatement) TokenLiteral() string { return wfs.Token.Literal }

// ... (Depois de GetByteAtExpression)

// --- NOVO NÓ DA AST (V14) ---

// ReturnStatement representa a instrução 'return <valor>'
// Implementa a interface 'Statement'.
type ReturnStatement struct {
	Token       lexer.Token // O token TOKEN_RETURN
	ReturnValue Expression  // A expressão do valor a ser retornado
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }

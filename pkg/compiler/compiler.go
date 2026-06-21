package compiler

import (
	"fmt"
	"ion-language/pkg/lexer"
	"ion-language/pkg/parser"
	"ion-language/pkg/vm"
)

// Type representa o tipo de uma expressão em tempo de compilação.
type Type int

const (
	TYPE_NUMBER   Type = iota
	TYPE_STRING
	TYPE_BOOLEAN
	TYPE_NIL
	TYPE_ARRAY
	TYPE_FUNCTION
	TYPE_UNKNOWN
)

func (t Type) String() string {
	switch t {
	case TYPE_NUMBER:
		return "number"
	case TYPE_STRING:
		return "string"
	case TYPE_BOOLEAN:
		return "boolean"
	case TYPE_NIL:
		return "nil"
	case TYPE_ARRAY:
		return "array"
	case TYPE_FUNCTION:
		return "function"
	default:
		return "unknown"
	}
}

// Symbol representa uma variável (local ou global).
type Symbol struct {
	Name    string
	Scope   string // "global", "local"
	Index   int    // Índice no slot local ou no pool de globais
	VarType Type   // Tipo da variável para type checking
}

// SymbolTable rastreia todas as variáveis em um escopo.
type SymbolTable struct {
	store map[string]Symbol
	Outer *SymbolTable // Para escopos aninhados (blocos)

	localCount int
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		store: make(map[string]Symbol),
	}
}

func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	s := NewSymbolTable()
	s.Outer = outer
	return s
}

// Define registra um novo símbolo local (parâmetro).
func (s *SymbolTable) Define(name string, scope string, index int, varType Type) Symbol {
	symbol := Symbol{Name: name, Scope: scope, Index: index, VarType: varType}
	s.store[name] = symbol
	return symbol
}

// Resolve encontra um símbolo pelo nome, percorrendo escopos aninhados.
func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	symbol, ok := s.store[name]
	if !ok && s.Outer != nil {
		return s.Outer.Resolve(name)
	}
	return symbol, ok
}

// --- COMPILADOR REATORADO (V9) ---
type Compiler struct {
	function *vm.FunctionObject

	symbolTable *SymbolTable

	// --- MUDANÇAS V13 ---
	scopeDepth int // 0 é global, 1 é <main>, 2+ é função
	localCount int // Quantos locais *esta* função possui
	// --- FIM V13 ---

	globals     map[string]int
	errors      []string
	typeStack    []Type
	loopStack   []int    // endereços de início de loop para continue
	breakStack  [][]int  // endereços de jump para resolver no break
}

func (c *Compiler) pushType(t Type) {
	c.typeStack = append(c.typeStack, t)
}

func (c *Compiler) popType() Type {
	if len(c.typeStack) == 0 {
		return TYPE_UNKNOWN
	}
	t := c.typeStack[len(c.typeStack)-1]
	c.typeStack = c.typeStack[:len(c.typeStack)-1]
	return t
}

func (c *Compiler) peekType(distance int) Type {
	if len(c.typeStack)-1-distance < 0 {
		return TYPE_UNKNOWN
	}
	return c.typeStack[len(c.typeStack)-1-distance]
}

func (c *Compiler) discardType() {
	if len(c.typeStack) > 0 {
		c.typeStack = c.typeStack[:len(c.typeStack)-1]
	}
}

func (c *Compiler) getTypeFromTypeNode(node parser.Expression) Type {
	switch n := node.(type) {
	case *parser.TypeIdentifier:
		switch n.Token.Type {
		case lexer.TOKEN_NUMBER_TYPE:
			return TYPE_NUMBER
		case lexer.TOKEN_STRING_TYPE:
			return TYPE_STRING
		case lexer.TOKEN_BOOLEAN_TYPE:
			return TYPE_BOOLEAN
		}
	case *parser.ArrayTypeNode:
		return TYPE_ARRAY
	}
	return TYPE_UNKNOWN
}

// NewCompiler cria um *novo* compilador (para o script ou uma função).
func NewCompiler(globals map[string]int) *Compiler {
	name := "<script>"

	return &Compiler{
		function: &vm.FunctionObject{
			Arity: 0,
			Chunk: vm.NewChunk(),
			Name:  name,
		},
		symbolTable: NewSymbolTable(),

		scopeDepth: 0,
		localCount: 0,

		globals:     globals,
		errors:      []string{},
		typeStack:   []Type{},
		loopStack:   []int{},
		breakStack:  [][]int{},
	}
}

// Compile é o ponto de entrada principal.
// Retorna o FunctionObject (o script 'main')
func Compile(program *parser.Program) (*vm.FunctionObject, []string) {
	globals := make(map[string]int)

	c := NewCompiler(globals)

	c.symbolTable.Define(c.function.Name, "local", c.localCount, TYPE_FUNCTION)
	c.localCount++

	if err := c.compileStatements(program.Statements); err != nil {
		c.addError(err)
	}

	mainFunc := c.emitReturn()

	if len(c.errors) > 0 {
		return nil, c.errors
	}

	return mainFunc, nil
}

// --- Roteadores (Atualizados) ---

func (c *Compiler) compileStatements(stmts []parser.Statement) error {
	for _, stmt := range stmts {
		if err := c.compileStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (c *Compiler) compileStatement(stmt parser.Statement) error {
	switch s := stmt.(type) {
	// --- NOVOS CASES V9 ---
	case *parser.FunctionStatement:
		return c.compileFunctionStatement(s)
	case *parser.ExpressionStatement:
		// Usado para chamadas de função (ex: Saudacao("Ion"))
		if err := c.compileExpression(s.Expression); err != nil {
			return err
		}
		// O resultado da chamada fica na pilha, damos POP para descartar
		c.emitByte(byte(vm.OP_POP), s.Token.Line)
		return nil
		// --- FIM V9 ---
	case *parser.ReturnStatement:
		return c.compileReturnStatement(s)
	case *parser.DeclareStatement:
		return c.compileDeclareStatement(s)
	case *parser.AssignmentStatement:
		return c.compileAssignmentStatement(s)
	case *parser.DisplayStatement:
		return c.compileDisplayStatement(s)
	case *parser.ForStatement:
		return c.compileForStatement(s)
	case *parser.IfStatement:
		return c.compileIfStatement(s)
	case *parser.WhileStatement:
		return c.compileWhileStatement(s)
	case *parser.BreakStatement:
		return c.compileBreakStatement(s)
	case *parser.ContinueStatement:
		return c.compileContinueStatement(s)
	case *parser.ExitStatement:
		return c.compileExitStatement(s)
	case *parser.WriteFileStatement:
		return c.compileWriteFileStatement(s)
	default:
		return fmt.Errorf("compilador V9 não suporta o 'Statement' tipo %T", stmt)
	}
}

func (c *Compiler) compileReturnStatement(stmt *parser.ReturnStatement) error {
	line := stmt.Token.Line

	// 1. Compila a expressão do valor de retorno (ex: a + b)
	// (O resultado, ex: 15, estará no topo da pilha)
	if err := c.compileExpression(stmt.ReturnValue); err != nil {
		return err
	}

	// 2. Emite OP_RETURN
	// A VM (V9) irá:
	//    - Pop o valor (15) da pilha
	//    - Fechar o CallFrame atual
	//    - Push o valor (15) na pilha do chamador
	c.emitByte(byte(vm.OP_RETURN), line)
	return nil
}

func (c *Compiler) compileExpression(expr parser.Expression) error {
	switch e := expr.(type) {
	// --- NOVOS CASES V9 ---
	case *parser.CallExpression:
		return c.compileCallExpression(e)
	// --- FIM V9 ---

	case *parser.Identifier:
		return c.compileIdentifier(e)

	// (Restante dos cases não muda)
	case *parser.NumberLiteral:
		c.pushType(TYPE_NUMBER)
		val := vm.NewNumberValue(e.Value)
		return c.currentChunk().WriteConstant(val, e.Token.Line)
	case *parser.StringLiteral:
		c.pushType(TYPE_STRING)
		val := vm.NewStringValue(e.Value)
		return c.currentChunk().WriteConstant(val, e.Token.Line)
	case *parser.InfixExpression:
		return c.compileInfixExpression(e)
	case *parser.BooleanLiteral:
		c.pushType(TYPE_BOOLEAN)
		if e.Value {
			c.emitByte(byte(vm.OP_TRUE), e.Token.Line)
		} else {
			c.emitByte(byte(vm.OP_FALSE), e.Token.Line)
		}
		return nil
	case *parser.NilLiteral:
		c.pushType(TYPE_NIL)
		c.emitByte(byte(vm.OP_NIL), e.Token.Line)
		return nil
	case *parser.InputExpression:
		return c.compileInputExpression(e)
	case *parser.PrefixExpression:
		return c.compilePrefixExpression(e)
	case *parser.LogicalExpression:
		return c.compileLogicalExpression(e)
	case *parser.IndexExpression:
		return c.compileIndexExpression(e)
	case *parser.CharExpression:
		return c.compileCharExpression(e)
	case *parser.OrdExpression:
		return c.compileOrdExpression(e)
	case *parser.LenExpression:
		return c.compileLenExpression(e)
	case *parser.GetByteAtExpression:
		return c.compileGetByteAtExpression(e)
	case *parser.ToStringExpression:
		return c.compileToStringExpression(e)
	case *parser.ToNumberExpression:
		return c.compileToNumberExpression(e)
	case *parser.ReadFileExpression:
		return c.compileReadFileExpression(e)
	default:
		return fmt.Errorf("compilador V9 não suporta a 'Expression' tipo %T", expr)
	}
}

func (c *Compiler) compileLenExpression(expr *parser.LenExpression) error {
	if err := c.compileExpression(expr.Argument); err != nil {
		return err
	}
	argType := c.popType()
	if argType != TYPE_STRING && argType != TYPE_ARRAY && argType != TYPE_UNKNOWN {
		return fmt.Errorf("'len' requer string ou array, recebeu %s (Linha: %d)", argType, expr.Token.Line)
	}
	c.pushType(TYPE_NUMBER)
	c.emitByte(byte(vm.OP_LEN), expr.Token.Line)
	return nil
}

func (c *Compiler) compileGetByteAtExpression(expr *parser.GetByteAtExpression) error {
	if err := c.compileExpression(expr.Target); err != nil {
		return err
	}
	targetType := c.popType()
	if targetType != TYPE_STRING && targetType != TYPE_UNKNOWN {
		return fmt.Errorf("'get_byte_at' requer uma string, recebeu %s (Linha: %d)", targetType, expr.Token.Line)
	}

	if err := c.compileExpression(expr.Index); err != nil {
		return err
	}
	indexType := c.popType()
	if indexType != TYPE_NUMBER && indexType != TYPE_UNKNOWN {
		return fmt.Errorf("'get_byte_at' requer um número como índice, recebeu %s (Linha: %d)", indexType, expr.Token.Line)
	}

	c.pushType(TYPE_NUMBER)
	c.emitByte(byte(vm.OP_GET_BYTE_AT), expr.Token.Line)
	return nil
}

func (c *Compiler) compileDeclareStatement(stmt *parser.DeclareStatement) error {
	varName := stmt.Name.Value
	line := stmt.Token.Line

	declaredType := c.getTypeFromTypeNode(stmt.TypeNode)

	if stmt.Value != nil {
		if err := c.compileExpression(stmt.Value); err != nil {
			return err
		}
		valueType := c.popType()
		if valueType != TYPE_UNKNOWN && valueType != TYPE_NIL && declaredType != TYPE_UNKNOWN && valueType != declaredType {
			return fmt.Errorf("tipo incompatível na declaração de '%s': esperava %s, recebeu %s (Linha: %d)",
				varName, declaredType, valueType, line)
		}
	} else {
		if err := c.compileTypeNode(stmt.TypeNode, stmt.Token.Line); err != nil {
			return err
		}
		c.discardType()
	}

	var symbol Symbol
	var idx int
	var isLocal = c.scopeDepth > 0

	if isLocal {
		symbol = c.symbolTable.Define(varName, "local", c.localCount, declaredType)
		c.localCount++
		if symbol.Index > 255 {
			return fmt.Errorf("limite de 256 variáveis locais excedido na declaração de '%s' (Linha: %d)", varName, line)
		}
		c.emitBytes(byte(vm.OP_SET_LOCAL), byte(symbol.Index), line)
	} else {
		if _, exists := c.globals[varName]; exists {
			return fmt.Errorf("variável global '%s' já foi declarada (Linha: %d)", varName, line)
		}
		nameConst := vm.NewStringValue(varName)
		idx = c.currentChunk().AddConstant(nameConst)
		if idx > 255 {
			return fmt.Errorf("limite de 256 variáveis globais excedido")
		}
		c.globals[varName] = idx
		c.emitBytes(byte(vm.OP_DEFINE_GLOBAL), byte(idx), line)
	}

	return nil
}

func (c *Compiler) compileTypeNode(node parser.Expression, line int) error {
	switch n := node.(type) {

	case *parser.TypeIdentifier:
		switch n.Token.Type {
		case lexer.TOKEN_NUMBER_TYPE:
			c.pushType(TYPE_NUMBER)
			val := vm.NewNumberValue(0)
			return c.currentChunk().WriteConstant(val, line)
		case lexer.TOKEN_STRING_TYPE:
			c.pushType(TYPE_STRING)
			val := vm.NewStringValue("")
			return c.currentChunk().WriteConstant(val, line)
		case lexer.TOKEN_BOOLEAN_TYPE:
			c.pushType(TYPE_BOOLEAN)
			c.emitByte(byte(vm.OP_FALSE), line)
			return nil
		}

	case *parser.ArrayTypeNode:
		tempTypeNode := &parser.TypeIdentifier{Token: n.BaseType}
		if err := c.compileTypeNode(tempTypeNode, line); err != nil {
			return err
		}
		if err := c.compileExpression(n.Size); err != nil {
			return err
		}
		c.discardType()
		c.discardType()
		c.pushType(TYPE_ARRAY)
		c.emitByte(byte(vm.OP_NEW_ARRAY), line)
		return nil
	}

	return fmt.Errorf("tipo desconhecido encontrado pelo compilador (Linha: %d)", line)
}

func (c *Compiler) compileIndexExpression(expr *parser.IndexExpression) error {
	if err := c.compileExpression(expr.Left); err != nil {
		return err
	}
	arrayType := c.peekType(0)
	if arrayType != TYPE_ARRAY && arrayType != TYPE_UNKNOWN {
		return fmt.Errorf("índice só pode ser aplicado a arrays, recebeu %s (Linha: %d)", arrayType, expr.Token.Line)
	}
	if err := c.compileExpression(expr.Index); err != nil {
		return err
	}
	c.discardType()
	c.discardType()
	c.pushType(TYPE_UNKNOWN)
	c.emitByte(byte(vm.OP_GET_INDEX), expr.Token.Line)
	return nil
}

func (c *Compiler) compileAssignmentStatement(stmt *parser.AssignmentStatement) error {
	line := stmt.Token.Line
	isCompound := stmt.Operator != ":="

	switch left := stmt.Left.(type) {

	case *parser.Identifier:
		varName := left.Value

		if isCompound {
			if symbol, ok := c.symbolTable.Resolve(varName); ok {
				if symbol.Index > 255 {
					return fmt.Errorf("limite de 256 variáveis locais excedido (Linha: %d)", line)
				}
				c.emitBytes(byte(vm.OP_GET_LOCAL), byte(symbol.Index), line)
			} else {
				idx, ok := c.globals[varName]
				if !ok {
					return fmt.Errorf("variável '%s' não declarada (Linha: %d)", varName, line)
				}
				c.emitBytes(byte(vm.OP_GET_GLOBAL), byte(idx), line)
			}
		}

		if err := c.compileExpression(stmt.Value); err != nil {
			return err
		}
		valueType := c.popType()

		if symbol, ok := c.symbolTable.Resolve(varName); ok {
			if symbol.VarType != TYPE_UNKNOWN && valueType != TYPE_UNKNOWN && valueType != TYPE_NIL && symbol.VarType != valueType {
				return fmt.Errorf("tipo incompatível na atribuição a '%s': esperava %s, recebeu %s (Linha: %d)",
					varName, symbol.VarType, valueType, line)
			}
			if isCompound {
				if valueType != TYPE_NUMBER && valueType != TYPE_UNKNOWN {
					return fmt.Errorf("operador '%s' requer operandos número, recebeu %s (Linha: %d)",
						stmt.Operator, valueType, line)
				}
				c.emitCompoundOp(stmt.Operator, line)
			}
			c.pushType(valueType)
			if symbol.Index > 255 {
				return fmt.Errorf("limite de 256 variáveis locais excedido (Linha: %d)", line)
			}
			c.emitBytes(byte(vm.OP_SET_LOCAL), byte(symbol.Index), line)
			c.emitByte(byte(vm.OP_POP), line)

		} else {
			idx, ok := c.globals[varName]
			if !ok {
				return fmt.Errorf("variável '%s' não declarada (Linha: %d)", varName, line)
			}
			if isCompound {
				c.emitCompoundOp(stmt.Operator, line)
			}
			c.pushType(TYPE_UNKNOWN)
			c.emitBytes(byte(vm.OP_SET_GLOBAL), byte(idx), line)
		}

	case *parser.IndexExpression:
		// Caso: tape[10] := 123
		// (Ordem V10.1: array, index, valor)

		// 2a. Compila o array (ex: 'tape')
		if err := c.compileExpression(left.Left); err != nil {
			return err
		}

		// 2b. Compila o índice (ex: '10')
		if err := c.compileExpression(left.Index); err != nil {
			return err
		}

		// 2c. Compila o valor (ex: '123')
		if err := c.compileExpression(stmt.Value); err != nil {
			return err
		}

		// 2d. Emite OP_SET_INDEX
		// (OP_SET_INDEX usa 'pop()' 3x na VM, não precisa de OP_POP)
		c.emitByte(byte(vm.OP_SET_INDEX), line)

	default:
		return fmt.Errorf("lado esquerdo inválido para atribuição (Linha: %d)", line)
	}

	return nil
}

func (c *Compiler) emitCompoundOp(op string, line int) error {
	switch op {
	case "+=":
		c.emitByte(byte(vm.OP_ADD), line)
	case "-=":
		c.emitByte(byte(vm.OP_SUBTRACT), line)
	case "*=":
		c.emitByte(byte(vm.OP_MULTIPLY), line)
	case "/=":
		c.emitByte(byte(vm.OP_DIVIDE), line)
	default:
		return fmt.Errorf("operador composto desconhecido: %s (Linha: %d)", op, line)
	}
	return nil
}

// (Display, For, If, While: O código não muda, pois eles
// apenas chamam 'compileStatement' e 'compileExpression'
// que agora estão cientes do escopo)

// --- NOVAS FUNÇÕES DE COMPILAÇÃO V9 ---

// compileFunctionStatement (VERSÃO V13.1 - CORRIGIDA)
func (c *Compiler) compileFunctionStatement(stmt *parser.FunctionStatement) error {
	varName := stmt.Name.Value
	line := stmt.Token.Line

	// ... (Compila a função, V9) ...
	funcCompiler := NewCompiler(c.globals)
	funcCompiler.scopeDepth = 1
	funcCompiler.function.Name = varName
	funcCompiler.function.Arity = len(stmt.Parameters)
	funcCompiler.symbolTable.Define(varName, "local", funcCompiler.localCount, TYPE_FUNCTION)
	funcCompiler.localCount++
	for _, param := range stmt.Parameters {
		paramType := TYPE_UNKNOWN
		if param.Type != nil {
			paramType = c.getTypeFromTypeNode(param.Type)
		}
		funcCompiler.symbolTable.Define(param.Name.Value, "local", funcCompiler.localCount, paramType)
		funcCompiler.localCount++
	}
	if err := funcCompiler.compileStatements(stmt.Body); err != nil {
		return err
	}
	function := funcCompiler.emitReturn()

	// 5. Adiciona o FunctionObject ao pool de constantes do *compilador principal*
	idx := c.currentChunk().AddConstant(vm.NewFunctionValue(function))
	if idx > 255 {
		return fmt.Errorf("limite de 256 constantes excedido")
	}
	c.emitBytes(byte(vm.OP_CONSTANT), byte(idx), line)

	// 6. Define a função como uma variável (Global ou Local)
	if c.scopeDepth > 0 {
		symbol := c.symbolTable.Define(varName, "local", c.localCount, TYPE_FUNCTION)
		c.localCount++
		if symbol.Index > 255 {
			return fmt.Errorf("limite de 256 variáveis locais excedido (Linha: %d)", line)
		}
		c.emitBytes(byte(vm.OP_SET_LOCAL), byte(symbol.Index), line)
	} else {
		// É uma variável GLOBAL
		nameConst := vm.NewStringValue(varName)
		idx = c.currentChunk().AddConstant(nameConst)
		if idx > 255 {
			return fmt.Errorf("limite de 256 variáveis globais excedido")
		}
		c.globals[varName] = idx
		c.emitBytes(byte(vm.OP_DEFINE_GLOBAL), byte(idx), line)
	}
	return nil
}

func (c *Compiler) compileCallExpression(expr *parser.CallExpression) error {
	if err := c.compileExpression(expr.Function); err != nil {
		return err
	}
	for _, arg := range expr.Arguments {
		if err := c.compileExpression(arg); err != nil {
			return err
		}
	}
	c.discardType()
	for range expr.Arguments {
		c.discardType()
	}
	c.pushType(TYPE_UNKNOWN)
	line := expr.Token.Line
	c.emitBytes(byte(vm.OP_CALL), byte(len(expr.Arguments)), line)
	return nil
}

func (c *Compiler) compileIdentifier(expr *parser.Identifier) error {
	// AGORA PRECISA SABER O ESCOPO
	varName := expr.Value
	line := expr.Token.Line

	if symbol, ok := c.symbolTable.Resolve(varName); ok {
		c.pushType(symbol.VarType)
		if symbol.Index > 255 {
			return fmt.Errorf("limite de 256 variáveis locais excedido (Linha: %d)", line)
		}
		c.emitBytes(byte(vm.OP_GET_LOCAL), byte(symbol.Index), line)
	} else {
		idx, ok := c.globals[varName]
		if !ok {
			return fmt.Errorf("variável '%s' não declarada (Linha: %d)", varName, line)
		}
		c.pushType(TYPE_UNKNOWN)
		c.emitBytes(byte(vm.OP_GET_GLOBAL), byte(idx), line)
	}
	return nil
}

// --- Funções Auxiliares (Atualizadas V9) ---

func (c *Compiler) addError(e error) {
	c.errors = append(c.errors, e.Error())
}

// currentChunk retorna o chunk do *compilador atual*
func (c *Compiler) currentChunk() *vm.Chunk {
	return c.function.Chunk
}

func (c *Compiler) emitByte(b byte, line int) {
	c.currentChunk().WriteChunk(b, line)
}

func (c *Compiler) emitBytes(b1, b2 byte, line int) {
	c.emitByte(b1, line)
	c.emitByte(b2, line)
}

// emitReturn (NOVO V9) - Emite um retorno implícito (nil)
func (c *Compiler) emitReturn() *vm.FunctionObject {
	line := 1 // TODO: Rastrear linha

	// Retorno implícito de 'nil'
	c.emitByte(byte(vm.OP_NIL), line)
	c.emitByte(byte(vm.OP_RETURN), line)

	return c.function
}

// defineGlobal (Atualizado V9)
func (c *Compiler) defineGlobal(varName string, line int) error {
	if _, exists := c.globals[varName]; exists {
		// Permitir redeclaração global? Por enquanto, não.
		return fmt.Errorf("variável global '%s' já foi declarada (Linha: %d)", varName, line)
	}

	// Adiciona o NOME da variável ao pool de constantes do chunk atual
	nameConst := vm.NewStringValue(varName)
	nameIdx := c.currentChunk().AddConstant(nameConst)
	if nameIdx > 255 {
		return fmt.Errorf("limite de 256 variáveis globais excedido")
	}

	// Armazena o índice na tabela de globais
	c.globals[varName] = nameIdx

	c.emitBytes(byte(vm.OP_DEFINE_GLOBAL), byte(nameIdx), line)
	return nil
}

func (c *Compiler) defineLocal(name string) {
	symbol := c.symbolTable.Define(name, "local", c.localCount, TYPE_UNKNOWN)
	c.localCount++
	_ = symbol
}

// (O código de For, If, While, Infix, Input, Prefix, Logical não muda
// pois eles usam as funções de compilação que agora são cientes do escopo)

// (Cole as funções restantes de compilação aqui:
// compileDisplayStatement, compileForStatement, compileIfStatement,
// compileWhileStatement, compileInfixExpression, compileInputExpression,
// compilePrefixExpression, compileLogicalExpression)

// (E as funções auxiliares de jump:
// currentAddress, emitJump, patchJump, emitLoop, getStmtLine)

// --- COLE O RESTANTE DO SEU compiler.go ANTIGO AQUI ---
// (Certifique-se de que eles usem 'c.currentChunk()' em vez de 'c.chunk')

// Exemplo de como adaptar uma função antiga:
func (c *Compiler) compileDisplayStatement(stmt *parser.DisplayStatement) error {
	if len(stmt.Arguments) == 0 {
		return nil
	}
	if err := c.compileExpression(stmt.Arguments[0]); err != nil {
		return err
	}
	for i := 1; i < len(stmt.Arguments); i++ {
		if err := c.compileExpression(stmt.Arguments[i]); err != nil {
			return err
		}
		c.emitByte(byte(vm.OP_CONCAT), stmt.Token.Line)
	}
	c.emitByte(byte(vm.OP_DISPLAY), stmt.Token.Line)
	return nil
}

// compileForStatement (VERSÃO V13.2 - CORRIGIDA PARA ESCOPO)
// compileForStatement (VERSÃO V13.3 - CORRIGIDA PARA ESCOPO)
func (c *Compiler) compileForStatement(stmt *parser.ForStatement) error {
	// 1. Compila o valor inicial (ex: 1)
	if err := c.compileExpression(stmt.Start); err != nil {
		return err
	}

	varName := stmt.Counter.Value
	line := stmt.Token.Line

	// 2. Resolve a variável (local ou global?)
	symbol, isLocal := c.symbolTable.Resolve(varName)
	var globalIndex int
	var isGlobal bool

	if !isLocal {
		globalIndex, isGlobal = c.globals[varName]
		if !isGlobal {
			return fmt.Errorf("variável '%s' não declarada (Linha: %d)", varName, line)
		}
	}

	if isLocal {
		if symbol.Index > 255 {
			return fmt.Errorf("limite de 256 variáveis locais excedido (Linha: %d)", line)
		}
		c.emitBytes(byte(vm.OP_SET_LOCAL), byte(symbol.Index), line)
	} else {
		c.emitBytes(byte(vm.OP_SET_GLOBAL), byte(globalIndex), line)
	}

	if isLocal {
		c.emitByte(byte(vm.OP_POP), line)
	}

	loopStart := c.currentAddress()
	c.loopStack = append(c.loopStack, loopStart)
	c.breakStack = append(c.breakStack, []int{})

	if err := c.compileExpression(stmt.Counter); err != nil {
		return err
	}
	if err := c.compileExpression(stmt.End); err != nil {
		return err
	}
	stepNeg := false
	if lit, ok := stmt.Step.(*parser.NumberLiteral); ok && lit.Value < 0 {
		stepNeg = true
	}
	if stepNeg {
		c.emitByte(byte(vm.OP_LESS), line)
	} else {
		c.emitByte(byte(vm.OP_GREATER), line)
	}

	jumpToBody := c.emitJump(byte(vm.OP_JUMP_IF_FALSE), line)
	jumpToExit := c.emitJump(byte(vm.OP_JUMP), line)

	c.patchJump(jumpToBody)
	c.emitByte(byte(vm.OP_POP), line)

	// --- ESCOPO DE BLOCO ---
	oldLocalCount := c.localCount
	oldScope := c.scopeDepth
	c.scopeDepth++
	oldTable := c.symbolTable
	c.symbolTable = NewEnclosedSymbolTable(oldTable)
	for _, bodyStmt := range stmt.Body {
		if err := c.compileStatement(bodyStmt); err != nil {
			return err
		}
	}
	c.symbolTable = oldTable
	c.localCount = oldLocalCount
	c.scopeDepth = oldScope
	// --- FIM ESCOPO DE BLOCO ---
	if err := c.compileExpression(stmt.Counter); err != nil {
		return err
	}
	if err := c.compileExpression(stmt.Step); err != nil {
		return err
	}
	c.emitByte(byte(vm.OP_ADD), line)

	if isLocal {
		if symbol.Index > 255 {
			return fmt.Errorf("limite de 256 variáveis locais excedido (Linha: %d)", line)
		}
		c.emitBytes(byte(vm.OP_SET_LOCAL), byte(symbol.Index), line)
	} else {
		c.emitBytes(byte(vm.OP_SET_GLOBAL), byte(globalIndex), line)
	}

	if isLocal {
		c.emitByte(byte(vm.OP_POP), line)
	}

	c.emitLoop(loopStart, line)
	c.patchJump(jumpToExit)
	c.emitByte(byte(vm.OP_POP), line)

	breakAddrs := c.breakStack[len(c.breakStack)-1]
	exitAddr := c.currentAddress()
	for _, addr := range breakAddrs {
		jump := exitAddr - addr - 2
		if jump > 65535 {
			continue
		}
		c.currentChunk().Code[addr] = byte(jump>>8) & 0xFF
		c.currentChunk().Code[addr+1] = byte(jump) & 0xFF
	}
	c.loopStack = c.loopStack[:len(c.loopStack)-1]
	c.breakStack = c.breakStack[:len(c.breakStack)-1]

	return nil
}

// compileIfStatement (VERSÃO V13.6 - CORRIGIDA PARA STACK E SEM REGRESSÃO)
func (c *Compiler) compileIfStatement(stmt *parser.IfStatement) error {
	line := stmt.Token.Line

	if err := c.compileExpression(stmt.Condition); err != nil {
		return err
	}
	c.discardType()
	jumpToElse := c.emitJump(byte(vm.OP_JUMP_IF_FALSE), line)

	// 3. Bloco 'Then' (executa se for 'true')
	// --- CORREÇÃO V13.6 ---
	c.emitByte(byte(vm.OP_POP), line) // Pop o 'true' AQUI
	// --- FIM DA CORREÇÃO ---
	// --- ESCOPO DE BLOCO ---
	oldLocalCount := c.localCount
	oldScope := c.scopeDepth
	c.scopeDepth++
	oldTable := c.symbolTable
	c.symbolTable = NewEnclosedSymbolTable(oldTable)
	for _, s := range stmt.Consequence {
		if err := c.compileStatement(s); err != nil {
			return err
		}
	}
	c.symbolTable = oldTable
	c.localCount = oldLocalCount
	c.scopeDepth = oldScope
	// --- FIM ESCOPO DE BLOCO ---

	// O 'then' salta o 'else'
	jumpOverElse := c.emitJump(byte(vm.OP_JUMP), line)

	// 5. Ponto de 'Else' / Fim. Corrige o salto 'false'.
	c.patchJump(jumpToElse)

	// --- CORREÇÃO V13.6 ---
	c.emitByte(byte(vm.OP_POP), line) // Pop o 'false' AQUI
	// --- FIM DA CORREÇÃO ---

	if stmt.Alternative != nil {
		// 6. Compila o 'else'
		// --- ESCOPO DE BLOCO ---
		oldLocalCount2 := c.localCount
		oldScope2 := c.scopeDepth
		c.scopeDepth++
		oldTable2 := c.symbolTable
		c.symbolTable = NewEnclosedSymbolTable(oldTable2)
		for _, s := range stmt.Alternative {
			if err := c.compileStatement(s); err != nil {
				return err
			}
		}
		c.symbolTable = oldTable2
		c.localCount = oldLocalCount2
		c.scopeDepth = oldScope2
		// --- FIM ESCOPO DE BLOCO ---
	}

	// 7. Ponto final. Corrige o salto do 'then'
	c.patchJump(jumpOverElse)

	return nil
}

func (c *Compiler) compileBreakStatement(stmt *parser.BreakStatement) error {
	if len(c.loopStack) == 0 {
		return fmt.Errorf("'break' só pode ser usado dentro de um loop (Linha: %d)", stmt.Token.Line)
	}
	idx := len(c.breakStack) - 1
	addr := c.emitJump(byte(vm.OP_JUMP), stmt.Token.Line)
	c.breakStack[idx] = append(c.breakStack[idx], addr)
	return nil
}

func (c *Compiler) compileContinueStatement(stmt *parser.ContinueStatement) error {
	if len(c.loopStack) == 0 {
		return fmt.Errorf("'continue' só pode ser usado dentro de um loop (Linha: %d)", stmt.Token.Line)
	}
	loopStart := c.loopStack[len(c.loopStack)-1]
	c.emitLoop(loopStart, stmt.Token.Line)
	return nil
}

func (c *Compiler) compileWhileStatement(stmt *parser.WhileStatement) error {
	line := stmt.Token.Line
	loopStart := c.currentAddress()
	c.loopStack = append(c.loopStack, loopStart)
	c.breakStack = append(c.breakStack, []int{})
	if err := c.compileExpression(stmt.Condition); err != nil {
		return err
	}
	c.discardType()
	jumpToExit := c.emitJump(byte(vm.OP_JUMP_IF_FALSE), line)
	c.discardType()
	c.emitByte(byte(vm.OP_POP), line)
	// --- ESCOPO DE BLOCO ---
	oldLocalCount := c.localCount
	oldScope := c.scopeDepth
	c.scopeDepth++
	oldTable := c.symbolTable
	c.symbolTable = NewEnclosedSymbolTable(oldTable)
	for _, s := range stmt.Body {
		if err := c.compileStatement(s); err != nil {
			return err
		}
	}
	c.symbolTable = oldTable
	c.localCount = oldLocalCount
	c.scopeDepth = oldScope
	// --- FIM ESCOPO DE BLOCO ---
	c.emitLoop(loopStart, line)
	c.patchJump(jumpToExit)
	c.emitByte(byte(vm.OP_POP), line)

	breakAddrs := c.breakStack[len(c.breakStack)-1]
	exitAddr := c.currentAddress()
	for _, addr := range breakAddrs {
		jump := exitAddr - addr - 2
		if jump > 65535 {
			continue
		}
		c.currentChunk().Code[addr] = byte(jump>>8) & 0xFF
		c.currentChunk().Code[addr+1] = byte(jump) & 0xFF
	}
	c.loopStack = c.loopStack[:len(c.loopStack)-1]
	c.breakStack = c.breakStack[:len(c.breakStack)-1]

	return nil
}
func (c *Compiler) compileInfixExpression(expr *parser.InfixExpression) error {
	if err := c.compileExpression(expr.Left); err != nil {
		return err
	}
	if err := c.compileExpression(expr.Right); err != nil {
		return err
	}

	rightType := c.popType()
	leftType := c.popType()
	line := expr.Token.Line

	switch expr.Token.Type {
	case lexer.TOKEN_PLUS, lexer.TOKEN_MINUS, lexer.TOKEN_ASTERISK, lexer.TOKEN_SLASH, lexer.TOKEN_PERCENT:
		if leftType == TYPE_STRING && expr.Token.Type == lexer.TOKEN_PLUS {
			c.pushType(TYPE_STRING)
			c.emitByte(byte(vm.OP_CONCAT), line)
			return nil
		}
		if leftType != TYPE_NUMBER || rightType != TYPE_NUMBER {
			if leftType == TYPE_UNKNOWN || rightType == TYPE_UNKNOWN {
				c.pushType(TYPE_NUMBER)
			} else {
				return fmt.Errorf("operandos devem ser números para '%s', recebeu %s e %s (Linha: %d)",
					expr.Token.Literal, leftType, rightType, line)
			}
		} else {
			c.pushType(TYPE_NUMBER)
		}
		switch expr.Token.Type {
		case lexer.TOKEN_PLUS:
			c.emitByte(byte(vm.OP_ADD), line)
		case lexer.TOKEN_MINUS:
			c.emitByte(byte(vm.OP_SUBTRACT), line)
		case lexer.TOKEN_ASTERISK:
			c.emitByte(byte(vm.OP_MULTIPLY), line)
		case lexer.TOKEN_SLASH:
			c.emitByte(byte(vm.OP_DIVIDE), line)
		case lexer.TOKEN_PERCENT:
			c.emitByte(byte(vm.OP_MODULO), line)
		}

	case lexer.TOKEN_GREATER, lexer.TOKEN_LESS, lexer.TOKEN_GREATER_EQUAL, lexer.TOKEN_LESS_EQUAL:
		if leftType != TYPE_NUMBER || rightType != TYPE_NUMBER {
			if leftType == TYPE_UNKNOWN || rightType == TYPE_UNKNOWN {
				c.pushType(TYPE_BOOLEAN)
			} else {
				return fmt.Errorf("operandos devem ser números para '%s', recebeu %s e %s (Linha: %d)",
					expr.Token.Literal, leftType, rightType, line)
			}
		} else {
			c.pushType(TYPE_BOOLEAN)
		}
		switch expr.Token.Type {
		case lexer.TOKEN_GREATER:
			c.emitByte(byte(vm.OP_GREATER), line)
		case lexer.TOKEN_GREATER_EQUAL:
			c.emitByte(byte(vm.OP_GREATER_EQUAL), line)
		case lexer.TOKEN_LESS:
			c.emitByte(byte(vm.OP_LESS), line)
		case lexer.TOKEN_LESS_EQUAL:
			c.emitByte(byte(vm.OP_LESS_EQUAL), line)
		}

	case lexer.TOKEN_EQUAL_EQUAL, lexer.TOKEN_NOT_EQUAL:
		if leftType != TYPE_UNKNOWN && rightType != TYPE_UNKNOWN && leftType != rightType &&
			leftType != TYPE_NIL && rightType != TYPE_NIL {
			return fmt.Errorf("tipos incompatíveis para '%s': %s e %s (Linha: %d)",
				expr.Token.Literal, leftType, rightType, line)
		}
		c.pushType(TYPE_BOOLEAN)
		switch expr.Token.Type {
		case lexer.TOKEN_EQUAL_EQUAL:
			c.emitByte(byte(vm.OP_EQUAL), line)
		case lexer.TOKEN_NOT_EQUAL:
			c.emitByte(byte(vm.OP_NOT_EQUAL), line)
		}

	default:
		return fmt.Errorf("operador infix desconhecido: %s (Linha: %d)", expr.Token.Literal, line)
	}
	return nil
}
func (c *Compiler) compileInputExpression(expr *parser.InputExpression) error {
	if err := c.compileExpression(expr.Prompt); err != nil {
		return err
	}
	promptType := c.popType()
	if promptType != TYPE_STRING && promptType != TYPE_UNKNOWN {
		return fmt.Errorf("'input' requer uma string como prompt, recebeu %s (Linha: %d)", promptType, expr.Token.Line)
	}
	c.pushType(TYPE_STRING)
	c.emitByte(byte(vm.OP_INPUT), expr.Token.Line)
	return nil
}
func (c *Compiler) compilePrefixExpression(expr *parser.PrefixExpression) error {
	if err := c.compileExpression(expr.Right); err != nil {
		return err
	}
	line := expr.Token.Line
	switch expr.Token.Type {
	case lexer.TOKEN_NOT:
		c.discardType()
		c.pushType(TYPE_BOOLEAN)
		c.emitByte(byte(vm.OP_NOT), line)
	default:
		return fmt.Errorf("operador prefix desconhecido: %s (Linha: %d)", expr.Token.Literal, line)
	}
	return nil
}
func (c *Compiler) compileLogicalExpression(expr *parser.LogicalExpression) error {
	line := expr.Token.Line

	if err := c.compileExpression(expr.Left); err != nil {
		return err
	}
	c.discardType()
	c.pushType(TYPE_BOOLEAN)

	if expr.Operator == "and" {
		jumpToExit := c.emitJump(byte(vm.OP_JUMP_IF_FALSE), line)
		c.discardType()
		c.emitByte(byte(vm.OP_POP), line)
		if err := c.compileExpression(expr.Right); err != nil {
			return err
		}
		c.discardType()
		c.pushType(TYPE_BOOLEAN)
		c.patchJump(jumpToExit)
	} else if expr.Operator == "or" {
		jumpToRight := c.emitJump(byte(vm.OP_JUMP_IF_FALSE), line)
		jumpToExit := c.emitJump(byte(vm.OP_JUMP), line)
		c.patchJump(jumpToRight)
		c.discardType()
		c.emitByte(byte(vm.OP_POP), line)
		if err := c.compileExpression(expr.Right); err != nil {
			return err
		}
		c.discardType()
		c.pushType(TYPE_BOOLEAN)
		c.patchJump(jumpToExit)
	}
	return nil
}

// --- Funções Auxiliares de Bytecode (Jumps) ---
func (c *Compiler) currentAddress() int {
	return len(c.currentChunk().Code)
}
func (c *Compiler) emitJump(op byte, line int) int {
	c.emitByte(op, line)
	c.emitByte(0xFF, line)
	c.emitByte(0xFF, line)
	return c.currentAddress() - 2
}
func (c *Compiler) patchJump(offset int) {
	jump := c.currentAddress() - offset - 2
	if jump > 65535 {
		c.addError(fmt.Errorf("salto (jump) muito grande para a VM"))
		return
	}
	c.currentChunk().Code[offset] = byte(jump>>8) & 0xFF
	c.currentChunk().Code[offset+1] = byte(jump) & 0xFF
}
func (c *Compiler) emitLoop(loopStart int, line int) {
	c.emitByte(byte(vm.OP_LOOP), line)
	offset := c.currentAddress() - loopStart + 2
	if offset > 65535 {
		c.addError(fmt.Errorf("loop (jump) muito grande para a VM"))
		return
	}
	c.emitByte(byte(offset>>8)&0xFF, line)
	c.emitByte(byte(offset)&0xFF, line)
}

func (c *Compiler) compileCharExpression(expr *parser.CharExpression) error {
	if err := c.compileExpression(expr.Argument); err != nil {
		return err
	}
	argType := c.popType()
	if argType != TYPE_NUMBER && argType != TYPE_UNKNOWN {
		return fmt.Errorf("'char' requer um número, recebeu %s (Linha: %d)", argType, expr.Token.Line)
	}
	c.pushType(TYPE_STRING)
	c.emitByte(byte(vm.OP_CHAR), expr.Token.Line)
	return nil
}

func (c *Compiler) compileOrdExpression(expr *parser.OrdExpression) error {
	if err := c.compileExpression(expr.Argument); err != nil {
		return err
	}
	c.discardType()
	c.pushType(TYPE_NUMBER)
	c.emitByte(byte(vm.OP_ORD), expr.Token.Line)
	return nil
}

func (c *Compiler) compileToStringExpression(expr *parser.ToStringExpression) error {
	if err := c.compileExpression(expr.Argument); err != nil {
		return err
	}
	c.discardType()
	c.pushType(TYPE_STRING)
	c.emitByte(byte(vm.OP_TO_STRING), expr.Token.Line)
	return nil
}

func (c *Compiler) compileToNumberExpression(expr *parser.ToNumberExpression) error {
	if err := c.compileExpression(expr.Argument); err != nil {
		return err
	}
	c.discardType()
	c.pushType(TYPE_NUMBER)
	c.emitByte(byte(vm.OP_TO_NUMBER), expr.Token.Line)
	return nil
}

func (c *Compiler) compileExitStatement(stmt *parser.ExitStatement) error {
	if err := c.compileExpression(stmt.Code); err != nil {
		return err
	}
	c.discardType()
	c.emitByte(byte(vm.OP_EXIT), stmt.Token.Line)
	return nil
}

func (c *Compiler) compileReadFileExpression(expr *parser.ReadFileExpression) error {
	if err := c.compileExpression(expr.Path); err != nil {
		return err
	}
	pathType := c.popType()
	if pathType != TYPE_STRING && pathType != TYPE_UNKNOWN {
		return fmt.Errorf("'readFile' requer uma string como caminho, recebeu %s (Linha: %d)", pathType, expr.Token.Line)
	}
	c.pushType(TYPE_STRING)
	c.emitByte(byte(vm.OP_READ_FILE), expr.Token.Line)
	return nil
}

func (c *Compiler) compileWriteFileStatement(stmt *parser.WriteFileStatement) error {
	if err := c.compileExpression(stmt.Path); err != nil {
		return err
	}
	pathType := c.popType()
	if pathType != TYPE_STRING && pathType != TYPE_UNKNOWN {
		return fmt.Errorf("'writeFile' requer uma string como caminho, recebeu %s (Linha: %d)", pathType, stmt.Token.Line)
	}
	if err := c.compileExpression(stmt.Content); err != nil {
		return err
	}
	contentType := c.popType()
	if contentType != TYPE_STRING && contentType != TYPE_UNKNOWN {
		return fmt.Errorf("'writeFile' requer uma string como conteúdo, recebeu %s (Linha: %d)", contentType, stmt.Token.Line)
	}
	c.emitByte(byte(vm.OP_WRITE_FILE), stmt.Token.Line)
	return nil
}

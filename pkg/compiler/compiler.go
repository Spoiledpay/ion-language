package compiler

import (
	"fmt"
	"ion-language/pkg/lexer"
	"ion-language/pkg/parser"
	"ion-language/pkg/vm"
)

// --- NOVO STRUCT V9: RASTREADOR DE ESCOPO ---
// Symbol representa uma variável (local ou global).
type Symbol struct {
	Name  string
	Scope string // "global", "local"
	Index int    // Índice no slot local ou no pool de globais
}

// SymbolTable rastreia todas as variáveis em um escopo.
type SymbolTable struct {
	store map[string]Symbol
	// Para escopos aninhados (ainda não usado na V9, mas bom para o futuro)
	// Outer *SymbolTable

	// Contagem de variáveis locais neste escopo
	localCount int
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		store: make(map[string]Symbol),
		// 'Outer' não é necessário para o Escopo de Função (V13)
	}
}

// Define registra um novo símbolo local (parâmetro).
func (s *SymbolTable) Define(name string, scope string, index int) Symbol {
	symbol := Symbol{Name: name, Scope: scope, Index: index}
	s.store[name] = symbol
	return symbol
}

// Resolve encontra um símbolo pelo nome.
func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	symbol, ok := s.store[name]
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

	globals map[string]int
	errors  []string
}

// NewCompiler cria um *novo* compilador (para o script ou uma função).
func NewCompiler(globals map[string]int) *Compiler { // <--- MUDANÇA V13

	// (O resto da função é quase o mesmo)
	name := "<script>" // V13: Começa como script

	return &Compiler{
		function: &vm.FunctionObject{
			Arity: 0,
			Chunk: vm.NewChunk(),
			Name:  name,
		},
		symbolTable: NewSymbolTable(),

		// --- MUDANÇAS V13 ---
		scopeDepth: 0, // Começa no escopo 0 (global)
		localCount: 0, // Sem locais ainda
		// --- FIM V13 ---

		globals: globals,
		errors:  []string{},
	}
}

// Compile é o ponto de entrada principal.
// Retorna o FunctionObject (o script 'main')
func Compile(program *parser.Program) (*vm.FunctionObject, []string) {
	globals := make(map[string]int)

	c := NewCompiler(globals) // <--- MUDANÇA V13

	// --- MUDANÇAS V13 ---
	// Entra no escopo 1 (o script 'main')
	c.scopeDepth++
	// O slot 0 da pilha é reservado para a função do script
	c.symbolTable.Define(c.function.Name, "local", c.localCount)
	c.localCount++
	// --- FIM V13 ---

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
		val := vm.NewNumberValue(e.Value)
		return c.currentChunk().WriteConstant(val, e.Token.Line)
	case *parser.StringLiteral:
		val := vm.NewStringValue(e.Value)
		return c.currentChunk().WriteConstant(val, e.Token.Line)
	case *parser.InfixExpression:
		return c.compileInfixExpression(e)
	case *parser.BooleanLiteral:
		if e.Value {
			c.emitByte(byte(vm.OP_TRUE), e.Token.Line)
		} else {
			c.emitByte(byte(vm.OP_FALSE), e.Token.Line)
		}
		return nil
	case *parser.NilLiteral:
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
	default:
		return fmt.Errorf("compilador V9 não suporta a 'Expression' tipo %T", expr)
	}
}

// compileLenExpression gera bytecode para len(string_ou_array)
func (c *Compiler) compileLenExpression(expr *parser.LenExpression) error {
	// 1. Compila a expressão do argumento (a string ou array)
	// (O objeto estará no topo da pilha)
	if err := c.compileExpression(expr.Argument); err != nil {
		return err
	}

	// 2. Emite OP_LEN
	// A VM irá (pop objeto) e (push comprimento)
	c.emitByte(byte(vm.OP_LEN), expr.Token.Line)
	return nil
}

// compileGetByteAtExpression gera bytecode para get_byte_at(string, index)
func (c *Compiler) compileGetByteAtExpression(expr *parser.GetByteAtExpression) error {
	// 1. Compila a string alvo
	// (A string estará no topo da pilha)
	if err := c.compileExpression(expr.Target); err != nil {
		return err
	}

	// 2. Compila o índice
	// (Pilha agora: [ string, index ])
	if err := c.compileExpression(expr.Index); err != nil {
		return err
	}

	// 3. Emite OP_GET_BYTE_AT
	// A VM irá (pop index, pop string) e (push byte_num)
	c.emitByte(byte(vm.OP_GET_BYTE_AT), expr.Token.Line)
	return nil
}

// compileDeclareStatement (VERSÃO V13.1 - CORRIGIDA)
// compileDeclareStatement (VERSÃO V13.7 - CORRIGIDA PARA ESCOPO LOCAL)
func (c *Compiler) compileDeclareStatement(stmt *parser.DeclareStatement) error {
	varName := stmt.Name.Value
	line := stmt.Token.Line

	// 1. Registra a variável na SymbolTable.
	// (Fazemos isso ANTES de compilar o valor)
	var symbol Symbol
	var idx int
	var isLocal = c.scopeDepth > 0

	if isLocal {
		// Estamos em um escopo local (função ou main)
		symbol = c.symbolTable.Define(varName, "local", c.localCount)
		c.localCount++
	} else {
		// Estamos no escopo global
		if _, exists := c.globals[varName]; exists {
			return fmt.Errorf("variável global '%s' já foi declarada (Linha: %d)", varName, line)
		}
	}

	// 2. Compila o valor (se houver) ou o padrão
	if stmt.Value != nil {
		if err := c.compileExpression(stmt.Value); err != nil {
			return err
		}
	} else {
		if err := c.compileTypeNode(stmt.TypeNode, stmt.Token.Line); err != nil {
			return err
		}
	}

	// 3. Emite o opcode de definição
	if isLocal {
		// --- ESTA É A CORREÇÃO V13.7 ---
		// Emite OP_SET_LOCAL para *inicializar* a variável.
		// A VM (V13.3) usa 'peek()', então o valor permanece na pilha,
		// aumentando o stackTop (o que está correto para 'declare').
		c.emitBytes(byte(vm.OP_SET_LOCAL), byte(symbol.Index), line)
	} else {
		// Lógica do defineGlobal (V9)
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

// compileTypeNode (VERSÃO V13.1 - CORRIGIDA)
func (c *Compiler) compileTypeNode(node parser.Expression, line int) error {
	switch n := node.(type) {

	case *parser.TypeIdentifier:
		// Tipos simples (number, string, boolean)
		switch n.Token.Type {
		case lexer.TOKEN_NUMBER_TYPE:
			val := vm.NewNumberValue(0)
			return c.currentChunk().WriteConstant(val, line)
		case lexer.TOKEN_STRING_TYPE:
			val := vm.NewStringValue("")
			return c.currentChunk().WriteConstant(val, line)
		case lexer.TOKEN_BOOLEAN_TYPE:
			c.emitByte(byte(vm.OP_FALSE), line)
			return nil
		}

	case *parser.ArrayTypeNode:
		// Tipo Array: [tipo](tamanho)

		// --- INÍCIO DA CORREÇÃO V13.1 ---
		// 1. Determina o valor padrão para o tipo base e o coloca na pilha.
		// (Precisamos de um 'TypeIdentifier' temporário para chamar a nós mesmos)
		tempTypeNode := &parser.TypeIdentifier{Token: n.BaseType}
		if err := c.compileTypeNode(tempTypeNode, line); err != nil {
			return err
		}
		// Pilha agora: [ valor_padrao ]

		// 2. Compila a expressão de tamanho
		if err := c.compileExpression(n.Size); err != nil {
			return err
		}
		// Pilha agora: [ valor_padrao, tamanho ]

		// 3. Emite OP_NEW_ARRAY
		// A VM irá (pop tamanho, pop valor_padrao) e (push novo_array)
		c.emitByte(byte(vm.OP_NEW_ARRAY), line)
		return nil
		// --- FIM DA CORREÇÃO ---
	}

	return fmt.Errorf("tipo desconhecido encontrado pelo compilador (Linha: %d)", line)
}

// compileIndexExpression gera bytecode para *ler* um índice (ex: display tape[10])
func (c *Compiler) compileIndexExpression(expr *parser.IndexExpression) error {
	// 1. Compila o array (ex: 'tape')
	// (O array estará no topo da pilha)
	if err := c.compileExpression(expr.Left); err != nil {
		return err
	}

	// 2. Compila o índice (ex: '10')
	// (Pilha agora: [ array, index ])
	if err := c.compileExpression(expr.Index); err != nil {
		return err
	}

	// 3. Emite OP_GET_INDEX
	// A VM irá (pop index, pop array) e (push valor_do_indice)
	c.emitByte(byte(vm.OP_GET_INDEX), expr.Token.Line)
	return nil
}

func (c *Compiler) compileAssignmentStatement(stmt *parser.AssignmentStatement) error {
	line := stmt.Token.Line

	// Verifica o que está do lado esquerdo (Left)
	switch left := stmt.Left.(type) {

	case *parser.Identifier:
		// Caso: x := 123

		// 1. Compila o valor (ex: '123').
		if err := c.compileExpression(stmt.Value); err != nil {
			return err
		}

		varName := left.Value

		// 2. Resolve o escopo (Local ou Global?)
		if symbol, ok := c.symbolTable.Resolve(varName); ok {
			// É uma variável local (parâmetro)
			c.emitBytes(byte(vm.OP_SET_LOCAL), byte(symbol.Index), line)

			// --- CORREÇÃO V13.3 ---
			// Como SET_LOCAL usa 'peek', o valor ainda está na pilha.
			// A atribuição (:=) deve limpar a pilha.
			c.emitByte(byte(vm.OP_POP), line)
			// --- FIM DA CORREÇÃO ---

		} else {
			// É uma variável global
			idx, ok := c.globals[varName]
			if !ok {
				return fmt.Errorf("variável '%s' não declarada (Linha: %d)", varName, line)
			}
			c.emitBytes(byte(vm.OP_SET_GLOBAL), byte(idx), line)
			// (OP_SET_GLOBAL usa 'pop()' na VM, então não precisa de OP_POP aqui)
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
	funcCompiler.symbolTable.Define(varName, "local", funcCompiler.localCount)
	funcCompiler.localCount++
	for _, param := range stmt.Parameters {
		funcCompiler.symbolTable.Define(param.Value, "local", funcCompiler.localCount)
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
		// --- ESTA É A CORREÇÃO ---
		// É uma variável LOCAL
		symbol := c.symbolTable.Define(varName, "local", c.localCount)
		c.localCount++
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
	// 1. Compila a função (o nome, ex: 'Saudacao')
	// (Isso coloca o FunctionObject no topo da pilha)
	if err := c.compileExpression(expr.Function); err != nil {
		return err
	}

	// 2. Compila todos os argumentos
	for _, arg := range expr.Arguments {
		if err := c.compileExpression(arg); err != nil {
			return err
		}
	}

	// 3. Emite OP_CALL com o número de argumentos
	line := expr.Token.Line
	c.emitBytes(byte(vm.OP_CALL), byte(len(expr.Arguments)), line)
	return nil
}

func (c *Compiler) compileIdentifier(expr *parser.Identifier) error {
	// AGORA PRECISA SABER O ESCOPO
	varName := expr.Value
	line := expr.Token.Line

	// 1. Tenta resolver como local
	if symbol, ok := c.symbolTable.Resolve(varName); ok {
		c.emitBytes(byte(vm.OP_GET_LOCAL), byte(symbol.Index), line)
	} else {
		// 2. Se não, resolve como global
		idx, ok := c.globals[varName]
		if !ok {
			return fmt.Errorf("variável '%s' não declarada (Linha: %d)", varName, line)
		}
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
	// A V13 usa escopo de função, então não precisamos checar re-declaração
	// (permitimos sombreamento se implementarmos escopo de bloco)

	// O índice da nova variável é o 'localCount' atual
	symbol := c.symbolTable.Define(name, "local", c.localCount)
	c.localCount++

	// O compilador NÃO emite OP_SET_LOCAL.
	// O valor já está no topo da pilha, pronto para ser
	// o novo local [stackSlot + index]
	_ = symbol // (usamos a variável para evitar erro de 'unused')
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

	// 3. Emite o 'set' inicial (usando o escopo correto)
	if isLocal {
		c.emitBytes(byte(vm.OP_SET_LOCAL), byte(symbol.Index), line)
	} else {
		c.emitBytes(byte(vm.OP_SET_GLOBAL), byte(globalIndex), line)
	}

	// --- CORREÇÃO V13.3 (POP 1) ---
	// Limpa o valor de inicialização da pilha (se for local)
	if isLocal {
		c.emitByte(byte(vm.OP_POP), line)
	}
	// --- FIM DA CORREÇÃO ---

	// --- Loop ---
	loopStart := c.currentAddress()

	// 4. Compila a condição (ex: i > end)
	if err := c.compileExpression(stmt.Counter); err != nil {
		return err
	}
	if err := c.compileExpression(stmt.End); err != nil {
		return err
	}
	c.emitByte(byte(vm.OP_GREATER), line) // Condição de saída do For

	jumpToBody := c.emitJump(byte(vm.OP_JUMP_IF_FALSE), line)
	jumpToExit := c.emitJump(byte(vm.OP_JUMP), line)

	c.patchJump(jumpToBody)
	c.emitByte(byte(vm.OP_POP), line)

	// 5. Compila o Corpo
	for _, bodyStmt := range stmt.Body {
		if err := c.compileStatement(bodyStmt); err != nil {
			return err
		}
	}

	// 6. Compila o incremento (ex: i + step)
	if err := c.compileExpression(stmt.Counter); err != nil {
		return err
	}
	if err := c.compileExpression(stmt.Step); err != nil {
		return err
	}
	c.emitByte(byte(vm.OP_ADD), line)

	// 7. Emite o 'set' do incremento (usando o escopo correto)
	if isLocal {
		c.emitBytes(byte(vm.OP_SET_LOCAL), byte(symbol.Index), line)
	} else {
		c.emitBytes(byte(vm.OP_SET_GLOBAL), byte(globalIndex), line)
	}

	// --- CORREÇÃO V13.3 (POP 2) ---
	// Limpa o valor do incremento da pilha (se for local)
	if isLocal {
		c.emitByte(byte(vm.OP_POP), line)
	}
	// --- FIM DA CORREÇÃO ---

	// --- Fim do Loop ---
	c.emitLoop(loopStart, line)
	c.patchJump(jumpToExit)
	c.emitByte(byte(vm.OP_POP), line)

	return nil
}

// compileIfStatement (VERSÃO V13.6 - CORRIGIDA PARA STACK E SEM REGRESSÃO)
func (c *Compiler) compileIfStatement(stmt *parser.IfStatement) error {
	line := stmt.Token.Line

	// 1. Compila a Condição (ex: cmd == 62)
	// (Pilha: [..., true/false])
	if err := c.compileExpression(stmt.Condition); err != nil {
		return err
	}

	// 2. Salta para o 'else' (ou fim) se for falso
	jumpToElse := c.emitJump(byte(vm.OP_JUMP_IF_FALSE), line)

	// 3. Bloco 'Then' (executa se for 'true')
	// --- CORREÇÃO V13.6 ---
	c.emitByte(byte(vm.OP_POP), line) // Pop o 'true' AQUI
	// --- FIM DA CORREÇÃO ---
	for _, s := range stmt.Consequence {
		if err := c.compileStatement(s); err != nil {
			return err
		}
	}

	// O 'then' salta o 'else'
	jumpOverElse := c.emitJump(byte(vm.OP_JUMP), line)

	// 5. Ponto de 'Else' / Fim. Corrige o salto 'false'.
	c.patchJump(jumpToElse)

	// --- CORREÇÃO V13.6 ---
	c.emitByte(byte(vm.OP_POP), line) // Pop o 'false' AQUI
	// --- FIM DA CORREÇÃO ---

	if stmt.Alternative != nil {
		// 6. Compila o 'else'
		for _, s := range stmt.Alternative {
			if err := c.compileStatement(s); err != nil {
				return err
			}
		}
	}

	// 7. Ponto final. Corrige o salto do 'then'
	c.patchJump(jumpOverElse)

	return nil
}

func (c *Compiler) compileWhileStatement(stmt *parser.WhileStatement) error {
	line := stmt.Token.Line
	loopStart := c.currentAddress()
	if err := c.compileExpression(stmt.Condition); err != nil {
		return err
	}
	jumpToExit := c.emitJump(byte(vm.OP_JUMP_IF_FALSE), line)
	c.emitByte(byte(vm.OP_POP), line)
	for _, s := range stmt.Body {
		if err := c.compileStatement(s); err != nil {
			return err
		}
	}
	c.emitLoop(loopStart, line)
	c.patchJump(jumpToExit)
	c.emitByte(byte(vm.OP_POP), line)
	return nil
}
func (c *Compiler) compileInfixExpression(expr *parser.InfixExpression) error {
	if err := c.compileExpression(expr.Left); err != nil {
		return err
	}
	if err := c.compileExpression(expr.Right); err != nil {
		return err
	}
	line := expr.Token.Line
	switch expr.Token.Type {
	case lexer.TOKEN_GREATER:
		c.emitByte(byte(vm.OP_GREATER), line)
	case lexer.TOKEN_LESS:
		c.emitByte(byte(vm.OP_LESS), line)
	case lexer.TOKEN_EQUAL_EQUAL:
		c.emitByte(byte(vm.OP_EQUAL), line)
	case lexer.TOKEN_NOT_EQUAL:
		c.emitByte(byte(vm.OP_NOT_EQUAL), line)
	case lexer.TOKEN_PLUS:
		c.emitByte(byte(vm.OP_ADD), line)
	case lexer.TOKEN_MINUS:
		c.emitByte(byte(vm.OP_SUBTRACT), line)
	case lexer.TOKEN_ASTERISK:
		c.emitByte(byte(vm.OP_MULTIPLY), line)
	case lexer.TOKEN_SLASH:
		c.emitByte(byte(vm.OP_DIVIDE), line)
	default:
		return fmt.Errorf("operador infix desconhecido: %s (Linha: %d)", expr.Token.Literal, line)
	}
	return nil
}
func (c *Compiler) compileInputExpression(expr *parser.InputExpression) error {
	if err := c.compileExpression(expr.Prompt); err != nil {
		return err
	}
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
		c.emitByte(byte(vm.OP_NOT), line)
	default:
		return fmt.Errorf("operador prefix desconhecido: %s (Linha: %d)", expr.Token.Literal, line)
	}
	return nil
}
func (c *Compiler) compileLogicalExpression(expr *parser.LogicalExpression) error {
	line := expr.Token.Line
	if expr.Operator == "and" {
		if err := c.compileExpression(expr.Left); err != nil {
			return err
		}
		jumpToExit := c.emitJump(byte(vm.OP_JUMP_IF_FALSE), line)
		c.emitByte(byte(vm.OP_POP), line)
		if err := c.compileExpression(expr.Right); err != nil {
			return err
		}
		c.patchJump(jumpToExit)
	} else if expr.Operator == "or" {
		if err := c.compileExpression(expr.Left); err != nil {
			return err
		}
		jumpToRight := c.emitJump(byte(vm.OP_JUMP_IF_FALSE), line)
		jumpToExit := c.emitJump(byte(vm.OP_JUMP), line)
		c.patchJump(jumpToRight)
		c.emitByte(byte(vm.OP_POP), line)
		if err := c.compileExpression(expr.Right); err != nil {
			return err
		}
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
	// 1. Compila a expressão do argumento (ex: 65)
	// (O número estará no topo da pilha)
	if err := c.compileExpression(expr.Argument); err != nil {
		return err
	}

	// 2. Emite OP_CHAR
	// A VM irá (pop 65) e (push "A")
	c.emitByte(byte(vm.OP_CHAR), expr.Token.Line)
	return nil
}

// compileOrdExpression gera bytecode para ord()
func (c *Compiler) compileOrdExpression(expr *parser.OrdExpression) error {
	// 1. Emite OP_ORD
	// A VM irá pausar, ler um char, e (push 65)
	c.emitByte(byte(vm.OP_ORD), expr.Token.Line)
	return nil
}

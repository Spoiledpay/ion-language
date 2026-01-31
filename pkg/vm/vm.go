package vm

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	STACK_MAX  = 256
	FRAMES_MAX = 64 // Número máximo de chamadas de função aninhadas
)

// --- NOVO STRUCT V9 ---
// CallFrame representa uma única chamada de função em andamento.
type CallFrame struct {
	function  *FunctionObject // A função que está sendo chamada
	ip        int             // O ponteiro de instrução (IP) desta função
	stackSlot int             // O índice na pilha da VM onde os locais desta função começam
}

// --- VM ATUALIZADA V9 ---
type VM struct {
	// A Pilha de Chamadas (Call Stack)
	frames     [FRAMES_MAX]CallFrame
	frameCount int // Quantos CallFrames estão ativos

	// A Pilha de Valores
	stack    []Value
	stackTop int // Aponta para o local *acima* do topo

	// Variáveis Globais
	globals map[string]Value
	reader  *bufio.Reader
}

// NewVM cria uma nova instância da Máquina Virtual.
func NewVM() *VM {
	return &VM{
		frames:     [FRAMES_MAX]CallFrame{},
		frameCount: 0,
		stack:      make([]Value, STACK_MAX),
		stackTop:   0,
		globals:    make(map[string]Value),
		reader:     bufio.NewReader(os.Stdin),
	}
}

// --- NOVO PONTO DE ENTRADA V9 ---
// Interpret é o novo ponto de entrada. Ele configura o 'main' e inicia a VM.
func (vm *VM) Interpret(mainFunction *FunctionObject) error {
	// 1. Coloca a função 'main' na pilha
	vm.push(NewFunctionValue(mainFunction))

	// 2. Configura o primeiro CallFrame para o 'main'
	frame := &vm.frames[0]
	frame.function = mainFunction
	frame.ip = 0
	frame.stackSlot = 0 // O 'main' começa no slot 0

	vm.frameCount = 1
	vm.stackTop = 1 // A função 'main' está no slot 0

	// 3. Inicia o loop de execução
	return vm.Run()
}

// Run é o loop principal de execução da VM (AGORA USA CALLFRAMES).
func (vm *VM) Run() error {
	// Referência para o CallFrame atual
	var frame *CallFrame

	for {
		// --- LÓGICA DE FRAME ATUALIZADA ---
		if vm.frameCount == 0 {
			return nil // Sem frames, programa terminado
		}
		frame = &vm.frames[vm.frameCount-1] // Pega o frame do topo
		// --- FIM DA ATUALIZAÇÃO ---

		// Modo de depuração (opcional):
		// vm.debugTraceStack()

		instruction := vm.readByte(frame)
		switch instruction {

		case byte(OP_HALT):
			return nil // Sucesso (embora OP_RETURN seja o novo 'fim')

		// --- Opcodes de Constantes ---
		case byte(OP_CONSTANT):
			constant := vm.readConstant(frame)
			vm.push(constant)
		case byte(OP_TRUE):
			vm.push(NewBoolValue(true))
		case byte(OP_FALSE):
			vm.push(NewBoolValue(false))
		case byte(OP_NIL):
			vm.push(NewNilValue())
		case byte(OP_NEW_ARRAY):
			// Pilha (V13.1): [ valor_padrao, tamanho ]

			// 1. Pega o tamanho da pilha
			sizeVal := vm.pop()
			if !IsNumber(sizeVal) {
				return vm.runtimeError(frame, "tamanho do array deve ser um número")
			}
			size := int(AsNumber(sizeVal))
			if size < 0 {
				return vm.runtimeError(frame, "tamanho do array não pode ser negativo")
			}

			// 2. Pega o valor padrão da pilha
			defaultValue := vm.pop()

			// 3. Cria o array (ArrayObject)
			arrayObj := &ArrayObject{
				Values: make([]Value, size),
			}

			// 4. Inicializa todas as células com o valor padrão
			for i := 0; i < size; i++ {
				arrayObj.Values[i] = defaultValue
			}

			// 5. Empurra o novo array na pilha
			vm.push(NewArrayValue(arrayObj))
		case byte(OP_GET_INDEX):
			// Pilha: [ array, index ]
			indexVal := vm.pop()
			arrayVal := vm.pop()

			if !IsArray(arrayVal) {
				return vm.runtimeError(frame, "só é possível acessar índice de um array")
			}
			if !IsNumber(indexVal) {
				return vm.runtimeError(frame, "índice do array deve ser um número")
			}

			obj := AsArray(arrayVal)
			index := int(AsNumber(indexVal))

			// Verificação de limites
			if index < 0 || index >= len(obj.Values) {
				return vm.runtimeError(frame, "índice (%d) fora dos limites do array (tamanho %d)", index, len(obj.Values))
			}

			// Empurra o valor encontrado
			vm.push(obj.Values[index])
		case byte(OP_SET_INDEX):
			// Pilha: [ array, index, valor ]
			value := vm.pop()
			indexVal := vm.pop()
			arrayVal := vm.pop()

			if !IsArray(arrayVal) {
				return vm.runtimeError(frame, "só é possível definir índice de um array")
			}
			if !IsNumber(indexVal) {
				return vm.runtimeError(frame, "índice do array deve ser um número")
			}

			obj := AsArray(arrayVal)
			index := int(AsNumber(indexVal))

			// Verificação de limites
			if index < 0 || index >= len(obj.Values) {
				return vm.runtimeError(frame, "índice (%d) fora dos limites do array (tamanho %d)", index, len(obj.Values))
			}

			// Define o valor no índice
			obj.Values[index] = value

		// --- Opcodes de Variáveis Globais (Não mudam) ---
		case byte(OP_DEFINE_GLOBAL):
			nameIdx := vm.readByte(frame)
			name := AsString(frame.function.Chunk.Constants[nameIdx])
			vm.globals[name] = vm.pop()
		case byte(OP_GET_GLOBAL):
			nameIdx := vm.readByte(frame)
			name := AsString(frame.function.Chunk.Constants[nameIdx])
			val, ok := vm.globals[name]
			if !ok {
				return vm.runtimeError(frame, "variável global indefinida '%s'", name)
			}
			vm.push(val)
		case byte(OP_SET_GLOBAL):
			nameIdx := vm.readByte(frame)
			name := AsString(frame.function.Chunk.Constants[nameIdx])
			val := vm.pop()
			if _, ok := vm.globals[name]; !ok {
				return vm.runtimeError(frame, "variável global indefinida '%s'", name)
			}
			vm.globals[name] = val

		// --- NOVOS OPCODES V9 (Locais) ---
		case byte(OP_GET_LOCAL):
			// O operando é o índice *relativo* ao início do slot do frame
			slot := int(vm.readByte(frame))
			// Nós lemos do slot absoluto da pilha
			vm.push(vm.stack[frame.stackSlot+slot])

		case byte(OP_SET_LOCAL):
			// V13.3: Revertido para 'peek'.
			// 'declare' (que usa SET_LOCAL) deve aumentar a pilha.
			slot := int(vm.readByte(frame))
			vm.stack[frame.stackSlot+slot] = vm.peek(0)

		// --- NOVOS OPCODES V9 (Chamada e Retorno) ---
		case byte(OP_CALL):
			argCount := int(vm.readByte(frame))
			// A função está na pilha, *abaixo* dos seus argumentos
			functionVal := vm.peek(argCount)

			if !IsFunction(functionVal) {
				return vm.runtimeError(frame, "só é possível chamar funções")
			}
			function := AsFunction(functionVal)

			if function.Arity != argCount {
				return vm.runtimeError(frame, "esperava %d argumentos, mas recebeu %d",
					function.Arity, argCount)
			}

			if vm.frameCount == FRAMES_MAX {
				return vm.runtimeError(frame, "estouro da pilha de chamadas (stack overflow)")
			}

			// Prepara o NOVO frame
			newFrame := &vm.frames[vm.frameCount]
			newFrame.function = function
			newFrame.ip = 0
			// O slot do novo frame começa onde a função (e seus args) estão
			newFrame.stackSlot = vm.stackTop - argCount - 1

			vm.frameCount++
			// O loop 'for' agora usará este novo frame

		case byte(OP_RETURN):
			// O valor de retorno (ex: 'nil') está no topo da pilha
			returnValue := vm.pop()

			// Descarta o frame atual
			vm.frameCount--
			if vm.frameCount == 0 {
				// Retornamos do 'main', programa terminou
				vm.pop() // Pop o script 'main'
				return nil
			}

			// Limpa a pilha, removendo os locais e a função
			vm.stackTop = frame.stackSlot
			// Empurra o valor de retorno, para o chamador usá-lo
			vm.push(returnValue)

		// --- Opcodes de Ações ---
		case byte(OP_DISPLAY):
			PrintValue(vm.pop())
			fmt.Println()
		case byte(OP_INPUT):
			prompt := vm.pop()
			if !IsString(prompt) {
				return vm.runtimeError(frame, "prompt do 'input' deve ser uma string")
			}
			fmt.Print(AsString(prompt))
			//reader := bufio.NewReader(os.Stdin)
			inputText, err := vm.reader.ReadString('\n')
			if err != nil {
				vm.push(NewStringValue(""))
			} else {
				cleanedInput := strings.TrimRight(inputText, "\r\n")
				vm.push(NewStringValue(cleanedInput))
			}
		case byte(OP_CHAR):
			// Pega o número (ex: 65) da pilha
			val := vm.pop()
			if !IsNumber(val) {
				return vm.runtimeError(frame, "argumento 'char()' deve ser um número")
			}

			// Converte o número (código ASCII) para uma string de 1 caractere
			num := int(AsNumber(val))
			charStr := string(rune(num))

			// Empurra a string (ex: "A") na pilha
			vm.push(NewStringValue(charStr))

		case byte(OP_ORD):
			// 'ord()' agora usa o leitor bufferizado da VM
			// e pula os caracteres de quebra de linha.
			for {
				inputByte, err := vm.reader.ReadByte()
				if err != nil {
					vm.push(NewNilValue()) // EOF
					break
				}

				// Ignora \r (13) e \n (10)
				if inputByte != 13 && inputByte != 10 {
					vm.push(NewNumberValue(float64(inputByte)))
					break // Encontramos um caractere válido
				}
				// Se for \r ou \n, o loop continua e lê o próximo byte
			}
		case byte(OP_LEN):
			// Pega a string (ou array) da pilha
			val := vm.pop()

			if IsString(val) {
				length := len(AsString(val))
				vm.push(NewNumberValue(float64(length)))
			} else if IsArray(val) {
				length := len(AsArray(val).Values)
				vm.push(NewNumberValue(float64(length)))
			} else {
				return vm.runtimeError(frame, "'len()' só pode ser usado em string ou array")
			}

		case byte(OP_GET_BYTE_AT):
			// Pilha: [ string, index ]
			indexVal := vm.pop()
			targetVal := vm.pop()

			if !IsString(targetVal) {
				return vm.runtimeError(frame, "'get_byte_at' só pode ser usado em string")
			}
			if !IsNumber(indexVal) {
				return vm.runtimeError(frame, "índice 'get_byte_at' deve ser um número")
			}

			targetStr := AsString(targetVal)
			index := int(AsNumber(indexVal))

			// Verificação de limites
			if index < 0 || index >= len(targetStr) {
				return vm.runtimeError(frame, "índice (%d) fora dos limites da string (tamanho %d)", index, len(targetStr))
			}

			// Pega o byte (ASCII) e empurra como número
			byteVal := targetStr[index]
			vm.push(NewNumberValue(float64(byteVal)))
		// --- Opcodes Aritméticos / Lógicos ---
		case byte(OP_ADD):
			// (O código de OP_ADD, SUBTRACT, MULTIPLY, DIVIDE não muda)
			b := vm.pop()
			a := vm.pop()
			if !IsNumber(a) || !IsNumber(b) {
				return vm.runtimeError(frame, "operandos devem ser números para '+'")
			}
			vm.push(NewNumberValue(AsNumber(a) + AsNumber(b)))
		case byte(OP_SUBTRACT):
			b := vm.pop()
			a := vm.pop()
			if !IsNumber(a) || !IsNumber(b) {
				return vm.runtimeError(frame, "operandos devem ser números para '-'")
			}
			vm.push(NewNumberValue(AsNumber(a) - AsNumber(b)))
		case byte(OP_MULTIPLY):
			b := vm.pop()
			a := vm.pop()
			if !IsNumber(a) || !IsNumber(b) {
				return vm.runtimeError(frame, "operandos devem ser números para '*'")
			}
			vm.push(NewNumberValue(AsNumber(a) * AsNumber(b)))
		case byte(OP_DIVIDE):
			b := vm.pop()
			a := vm.pop()
			if !IsNumber(a) || !IsNumber(b) {
				return vm.runtimeError(frame, "operandos devem ser números para '/'")
			}
			if AsNumber(b) == 0 {
				return vm.runtimeError(frame, "divisão por zero")
			}
			vm.push(NewNumberValue(AsNumber(a) / AsNumber(b)))
		case byte(OP_CONCAT):
			b := vm.pop()
			a := vm.pop()
			sa := Stringify(a)
			sb := Stringify(b)
			vm.push(NewStringValue(sa + sb))
		case byte(OP_GREATER):
			b := vm.pop()
			a := vm.pop()
			if !IsNumber(a) || !IsNumber(b) {
				return vm.runtimeError(frame, "operandos devem ser números para '>'")
			}
			vm.push(NewBoolValue(AsNumber(a) > AsNumber(b)))
		case byte(OP_LESS):
			b := vm.pop()
			a := vm.pop()
			if !IsNumber(a) || !IsNumber(b) {
				return vm.runtimeError(frame, "operandos devem ser números para '<'")
			}
			vm.push(NewBoolValue(AsNumber(a) < AsNumber(b)))
		case byte(OP_EQUAL):
			b := vm.pop()
			a := vm.pop()
			if IsNil(a) && IsNil(b) {
				vm.push(NewBoolValue(true))
			} else if IsNil(a) || IsNil(b) {
				vm.push(NewBoolValue(false))
			} else {
				vm.push(NewBoolValue(a.As == b.As))
			}
		case byte(OP_NOT_EQUAL):
			b := vm.pop()
			a := vm.pop()
			if IsNil(a) && IsNil(b) {
				vm.push(NewBoolValue(false)) // nil != nil é false
			} else if IsNil(a) || IsNil(b) {
				vm.push(NewBoolValue(true)) // nil != (outra coisa) é true
			} else {
				vm.push(NewBoolValue(a.As != b.As))
			}
		case byte(OP_NOT):
			val := vm.pop()
			is_truthy := vm.isTruthy(val)
			vm.push(NewBoolValue(!is_truthy))

		// --- Opcodes de Controle de Fluxo (Jumps) ---
		case byte(OP_JUMP):
			offset := vm.readShort(frame)
			frame.ip += offset
		case byte(OP_JUMP_IF_FALSE):
			offset := vm.readShort(frame)
			if !vm.isTruthy(vm.peek(0)) {
				frame.ip += offset
			}
		case byte(OP_LOOP):
			offset := vm.readShort(frame)
			frame.ip -= offset

		// --- Opcodes de Pilha ---
		case byte(OP_POP):
			vm.pop()

		default:
			return vm.runtimeError(frame, "Opcode desconhecido %d", instruction)
		}
	}
}

// --- Funções Auxiliares da VM (ATUALIZADAS) ---

func (vm *VM) runtimeError(frame *CallFrame, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	// Pega a linha do chunk da *função atual*
	line := frame.function.Chunk.GetLine(frame.ip - 1)
	return fmt.Errorf("[Linha %d] Erro de Execução: %s", line, msg)
}

func (vm *VM) isTruthy(val Value) bool {
	if IsBool(val) && AsBool(val) == false {
		return false
	}
	if IsNil(val) {
		return false
	}
	return true
}

// --- Funções de Leitura de Bytecode (ATUALIZADAS) ---

func (vm *VM) readByte(frame *CallFrame) byte {
	b := frame.function.Chunk.Code[frame.ip]
	frame.ip++
	return b
}

func (vm *VM) readShort(frame *CallFrame) int {
	b1 := int(vm.readByte(frame))
	b2 := int(vm.readByte(frame))
	return (b1 << 8) | b2
}

func (vm *VM) readConstant(frame *CallFrame) Value {
	idx := vm.readByte(frame)
	return frame.function.Chunk.Constants[idx]
}

// --- Funções da Pilha (Stack) (Não mudam) ---

func (vm *VM) push(val Value) {
	if vm.stackTop >= STACK_MAX {
		panic("Stack overflow!")
	}
	vm.stack[vm.stackTop] = val
	vm.stackTop++
}

func (vm *VM) pop() Value {
	if vm.stackTop == 0 {
		panic("Stack underflow!")
	}
	vm.stackTop--
	return vm.stack[vm.stackTop]
}

func (vm *VM) peek(distance int) Value {
	if vm.stackTop-1-distance < 0 {
		panic("Stack underflow on peek!")
	}
	return vm.stack[vm.stackTop-1-distance]
}

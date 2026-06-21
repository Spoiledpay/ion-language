package vm

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const DefaultStackMax = 4096
const DefaultFramesMax = 256

type CallFrame struct {
	function  *FunctionObject
	ip        int
	stackSlot int
}

type VM struct {
	frames     []CallFrame
	frameCount int

	stack    []Value
	stackTop int

	globals map[string]Value
	reader  *bufio.Reader
}

func NewVM() *VM {
	return NewVMConfig(DefaultStackMax, DefaultFramesMax)
}

func NewVMConfig(stackMax, framesMax int) *VM {
	return &VM{
		frames:     make([]CallFrame, framesMax),
		frameCount: 0,
		stack:      make([]Value, stackMax),
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

// Run é o loop principal de execução da VM.
func (vm *VM) Run() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Erro Interno da VM: %v", r)
		}
	}()

	var frame *CallFrame

	for {
		if vm.frameCount == 0 {
			return nil
		}
		frame = &vm.frames[vm.frameCount-1]

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
			slot := int(vm.readByte(frame))
			absSlot := frame.stackSlot + slot
			if absSlot < 0 || absSlot >= len(vm.stack) {
				return vm.runtimeError(frame, "slot local %d fora dos limites", slot)
			}
			vm.push(vm.stack[absSlot])

		case byte(OP_SET_LOCAL):
			slot := int(vm.readByte(frame))
			absSlot := frame.stackSlot + slot
			if absSlot < 0 || absSlot >= len(vm.stack) {
				return vm.runtimeError(frame, "slot local %d fora dos limites", slot)
			}
			val := vm.peek(0)
			vm.stack[absSlot] = val

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

			if vm.frameCount >= len(vm.frames) {
				return vm.runtimeError(frame, "estouro da pilha de chamadas (stack overflow)")
			}

			stackSlot := vm.stackTop - argCount - 1
			if stackSlot < 0 {
				return vm.runtimeError(frame, "pilha corrompida na chamada de função")
			}
			newFrame := &vm.frames[vm.frameCount]
			newFrame.function = function
			newFrame.ip = 0
			newFrame.stackSlot = stackSlot

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
			val := vm.pop()
			if !IsString(val) {
				return vm.runtimeError(frame, "'ord' requer uma string como argumento")
			}
			s := AsString(val)
			if len(s) == 0 {
				vm.push(NewNilValue())
			} else {
				vm.push(NewNumberValue(float64(s[0])))
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
		case byte(OP_LESS_EQUAL):
			b := vm.pop()
			a := vm.pop()
			if !IsNumber(a) || !IsNumber(b) {
				return vm.runtimeError(frame, "operandos devem ser números para '<='")
			}
			vm.push(NewBoolValue(AsNumber(a) <= AsNumber(b)))
		case byte(OP_GREATER_EQUAL):
			b := vm.pop()
			a := vm.pop()
			if !IsNumber(a) || !IsNumber(b) {
				return vm.runtimeError(frame, "operandos devem ser números para '>='")
			}
			vm.push(NewBoolValue(AsNumber(a) >= AsNumber(b)))
		case byte(OP_MODULO):
			b := vm.pop()
			a := vm.pop()
			if !IsNumber(a) || !IsNumber(b) {
				return vm.runtimeError(frame, "operandos devem ser números para '%'")
			}
			if AsNumber(b) == 0 {
				return vm.runtimeError(frame, "módulo por zero")
			}
			vm.push(NewNumberValue(float64(int(AsNumber(a)) % int(AsNumber(b)))))
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
		case byte(OP_TO_STRING):
			val := vm.pop()
			vm.push(NewStringValue(Stringify(val)))
		case byte(OP_TO_NUMBER):
			val := vm.pop()
			if IsNumber(val) {
				vm.push(val)
			} else if IsString(val) {
				s := AsString(val)
				num, err := fmt.Sscanf(s, "%g", new(float64))
				if err != nil || num == 0 {
					vm.push(NewNilValue())
				} else {
					var f float64
					fmt.Sscanf(s, "%g", &f)
					vm.push(NewNumberValue(f))
				}
			} else {
				vm.push(NewNilValue())
			}
		case byte(OP_EXIT):
			codeVal := vm.pop()
			code := 0
			if IsNumber(codeVal) {
				code = int(AsNumber(codeVal))
			}
			os.Exit(code)
		case byte(OP_READ_FILE):
			pathVal := vm.pop()
			if !IsString(pathVal) {
				return vm.runtimeError(frame, "'readFile' requer uma string como caminho")
			}
			data, err := os.ReadFile(AsString(pathVal))
			if err != nil {
				vm.push(NewNilValue())
			} else {
				vm.push(NewStringValue(string(data)))
			}
		case byte(OP_WRITE_FILE):
			contentVal := vm.pop()
			pathVal := vm.pop()
			if !IsString(pathVal) || !IsString(contentVal) {
				return vm.runtimeError(frame, "'writeFile' requer duas strings (caminho, conteúdo)")
			}
			err := os.WriteFile(AsString(pathVal), []byte(AsString(contentVal)), 0644)
			if err != nil {
				return vm.runtimeError(frame, "erro ao escrever arquivo '%s': %s", AsString(pathVal), err)
			}
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
	line := 0
	if frame.ip > 0 {
		line = frame.function.Chunk.GetLine(frame.ip - 1)
	}
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
	if vm.stackTop >= len(vm.stack) {
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

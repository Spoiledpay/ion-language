package vm

// OpCode é um único byte que representa uma instrução da VM.
type OpCode byte

// Definição de todas as instruções (Opcodes) da VM Ion V1
const (
	// --- Opcodes de Constantes ---

	// OP_CONSTANT: Carrega uma constante (número, string) da "pool de constantes"
	//              para a pilha da VM.
	// Formato: [ OP_CONSTANT ] [ index_da_constante (byte) ]
	OP_CONSTANT OpCode = iota

	// --- NOVOS OPCODES V4 (Valores Literais) ---
	OP_TRUE  // Empurra 'true'
	OP_FALSE // Empurra 'false'
	OP_NIL   // Empurra 'nil'
	// --- FIM DOS NOVOS OPCODES ---

	OP_NEW_ARRAY // Cria um novo array
	OP_GET_INDEX // Pega o valor em array[index]
	OP_SET_INDEX // Define o valor em array[index]
	// OP_DEFINE_GLOBAL: Define uma nova variável global.
	// Formato: [ OP_DEFINE_GLOBAL ] [ index_nome_global (byte) ]
	//          (Pega o valor do topo da pilha)
	OP_DEFINE_GLOBAL

	// OP_GET_GLOBAL: Empurra o valor de uma variável global para a pilha.
	// Formato: [ OP_GET_GLOBAL ] [ index_nome_global (byte) ]
	OP_GET_GLOBAL

	OP_NOT_EQUAL // Compara B != A

	// OP_SET_GLOBAL: Define o valor de uma variável global EXISTENTE.
	// Formato: [ OP_SET_GLOBAL ] [ index_nome_global (byte) ]
	//          (Pega o valor do topo da pilha)
	OP_SET_GLOBAL

	OP_CHAR // Converte um número para um caractere (string)
	OP_ORD
	// --- Opcodes de Ações ---

	OP_LEN // Retorna o comprimento de uma string
	OP_GET_BYTE_AT
	// OP_DISPLAY: Retira (pop) um valor da pilha e o imprime no console.
	// Formato: [ OP_DISPLAY ]
	OP_DISPLAY

	OP_INPUT

	// OP_JUMP_IF_FALSE: Pula um número de instruções se o topo da pilha
	//                   for "falso" (usado para sair do loop).
	// Formato: [ OP_JUMP_IF_FALSE ] [ offset_alto (byte) ] [ offset_baixo (byte) ]
	OP_JUMP_IF_FALSE

	// OP_JUMP: Pula incondicionalmente um número de instruções.
	// Formato: [ OP_JUMP ] [ offset_alto (byte) ] [ offset_baixo (byte) ]
	OP_JUMP

	// OP_LOOP: Pula *para trás* um número de instruções (para continuar o loop).
	// Formato: [ OP_LOOP ] [ offset_alto (byte) ] [ offset_baixo (byte) ]

	OP_LOOP // TODO: OP_LOOP é mais complexo, vamos usar OP_JUMP por enquanto
	// Vamos simplificar a V1: OP_JUMP serve tanto para frente quanto para trás.

	// --- Opcodes Aritméticos/Lógicos ---
	// Para o 'for i = 1 to 10', precisamos de comparação (<=) e incremento (+ 1).

	// OP_ADD: Retira A e B da pilha, empurra A + B
	OP_ADD

	OP_SUBTRACT // Empurra A - B
	OP_MULTIPLY // Empurra A * B
	OP_DIVIDE   // Empurra A / B
	// OP_LESS_EQUAL: Retira B e A, empurra (A <= B)
	// Precisamos de booleanos! Vamos simplificar.
	// A V1 só precisa de '+' (para o 'step') e '<=' (para a condição 'to')
	// E 'display' com vírgula precisa de concatenação.

	// OP_CONCAT: Concatena duas strings. (Para 'display "Olá", nome')
	OP_CONCAT

	// OP_GREATER: Compara B > A (usado no loop 'for')
	OP_GREATER // Usado para (i > 10) -> sair do loop

	OP_LESS  // Compara B < A
	OP_LESS_EQUAL    // Compara B <= A
	OP_GREATER_EQUAL // Compara B >= A
	OP_MODULO        // A % B
	OP_EQUAL // Compara B == A

	// OP_POP: Retira (descarta) o valor do topo da pilha.
	OP_POP
	OP_NOT

	OP_GET_LOCAL // Pega uma variável local (parâmetro) da pilha
	OP_SET_LOCAL // Define uma variável local (parâmetro) na pilha
	OP_CALL      // Chama uma função
	OP_RETURN
	OP_TO_STRING // Converte topo da pilha para string
	OP_TO_NUMBER // Converte topo da pilha para number (ou nil)
	OP_EXIT      // Encerra com código de saída
	OP_READ_FILE // Lê arquivo, push conteúdo como string
	OP_WRITE_FILE // Escreve conteúdo em arquivo

	OP_HALT
)

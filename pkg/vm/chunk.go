package vm

import "fmt"

// Chunk é o contêiner para o bytecode e os dados associados.
type Chunk struct {
	// Code é o array de bytes de instruções da VM.
	// Inclui Opcodes e seus operandos (ex: índice da constante).
	Code []byte

	// Constants é a "piscina de constantes"
	// Armazena os valores literais (números, strings)
	Constants []Value

	// Lines mapeia cada byte em 'Code' para uma linha no código-fonte.
	// Essencial para relatórios de erro em tempo de execução.
	Lines []int
}

// NewChunk inicializa um novo Chunk vazio.
func NewChunk() *Chunk {
	return &Chunk{
		Code:      []byte{},
		Constants: []Value{},
		Lines:     []int{},
	}
}

// WriteChunk adiciona um byte ao chunk (seja um OpCode ou um operando).
// Também armazena a linha do código-fonte correspondente.
func (c *Chunk) WriteChunk(b byte, line int) {
	c.Code = append(c.Code, b)
	c.Lines = append(c.Lines, line)
}

// AddConstant adiciona um valor à piscina de constantes com deduplicação.
func (c *Chunk) AddConstant(val Value) int {
	for i, existing := range c.Constants {
		if val.Type == existing.Type && val.As == existing.As {
			return i
		}
	}
	c.Constants = append(c.Constants, val)
	return len(c.Constants) - 1
}

// WriteConstant é um método de ajuda para o Compilador.
// Ele lida com a lógica de adicionar a constante E escrever os
// opcodes OP_CONSTANT + índice.
// Retorna um erro se o índice for maior que 255 (limite de 1 byte).
func (c *Chunk) WriteConstant(val Value, line int) error {
	idx := c.AddConstant(val)

	if idx > 255 {
		// Nosso operando OP_CONSTANT é de apenas 1 byte (0-255).
		// Se tivermos mais de 256 constantes, precisamos de um opcode
		// OP_CONSTANT_16 (de 2 bytes). Por enquanto, é um erro.
		return fmt.Errorf("limite de 256 constantes excedido")
	}

	c.WriteChunk(byte(OP_CONSTANT), line)
	c.WriteChunk(byte(idx), line)
	return nil
}

// GetLine retorna a linha do código-fonte para um dado offset no bytecode.
func (c *Chunk) GetLine(offset int) int {
	// TODO: Implementar uma forma mais eficiente (ex: RLE - Run-Length Encoding)
	// Por enquanto, o mapeamento 1-para-1 funciona.
	if offset < len(c.Lines) {
		return c.Lines[offset]
	}
	return 0 // Deve ser tratado como um erro
}

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"ion-language/pkg/vm" // Importa nosso motor da VM
)

var MagicNumber = []byte{0x49, 0x4F, 0x4E, 0x43} // "IONC"

func main() {
	// --- ATUALIZAÇÃO V13.8: Prompt de Uso ---
	// 1. Verificar argumentos
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "LabsObjects (R) Ion Virtual Machine Versão 1.13")
		fmt.Fprintln(os.Stderr, "Copyright (C) LabsObjects. Todos os direitos reservados.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "uso:  ionv <arquivo.ionc>")
		os.Exit(0) // Sair sem erro
	}
	// --- FIM DA ATUALIZAÇÃO ---

	filePath := os.Args[1]

	// 2. Carregar o bytecode
	mainFunction, err := loadBytecode(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao carregar bytecode de %s: %s\n", filePath, err)
		os.Exit(1)
	}

	// 3. Inicializar e rodar a VM (BANNERS REMOVIDOS)
	v := vm.NewVM()
	err = v.Interpret(mainFunction)

	// 4. Tratar erros de runtime
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %s\n", err)
		os.Exit(1)
	}
}

// loadBytecode agora é um wrapper para readFunction
func loadBytecode(filePath string) (*vm.FunctionObject, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 1. Verificar Magic Number
	magic := make([]byte, 4)
	if _, err := io.ReadFull(file, magic); err != nil {
		return nil, fmt.Errorf("falha ao ler magic number: %w", err)
	}
	if !bytes.Equal(magic, MagicNumber) {
		return nil, fmt.Errorf("arquivo .ionc inválido (magic number incorreto)")
	}

	// 2. Desserializa recursivamente a função 'main'
	le := binary.LittleEndian
	return readFunction(file, le)
}

// --- NOVOS HELPERS V9 (DESSERIALIZAÇÃO) ---

// readFunction desserializa um FunctionObject (Nome, Aridade, Chunk)
func readFunction(file io.Reader, le binary.ByteOrder) (*vm.FunctionObject, error) {
	// 1. Lê Nome e Aridade
	name, err := readString(file, le)
	if err != nil {
		return nil, err
	}
	var arity int64
	if err := binary.Read(file, le, &arity); err != nil {
		return nil, err
	}

	funcObj := &vm.FunctionObject{
		Name:  name,
		Arity: int(arity),
		Chunk: vm.NewChunk(),
	}

	// 2. Lê a Piscina de Constantes do Chunk
	var constCount int64
	if err := binary.Read(file, le, &constCount); err != nil {
		return nil, err
	}

	for i := int64(0); i < constCount; i++ {
		// Lê o TIPO da constante
		var valType int8
		if err := binary.Read(file, le, &valType); err != nil {
			return nil, err
		}
		// Lê o DADO da constante
		switch vm.ValueType(valType) {
		case vm.VAL_NUMBER:
			var num float64
			if err := binary.Read(file, le, &num); err != nil {
				return nil, err
			}
			funcObj.Chunk.AddConstant(vm.NewNumberValue(num))
		case vm.VAL_STRING:
			s, err := readString(file, le)
			if err != nil {
				return nil, err
			}
			funcObj.Chunk.AddConstant(vm.NewStringValue(s))
		case vm.VAL_FUNCTION:
			// É uma função aninhada! Desserializa recursivamente.
			nestedFunc, err := readFunction(file, le)
			if err != nil {
				return nil, err
			}
			funcObj.Chunk.AddConstant(vm.NewFunctionValue(nestedFunc))
		default:
			return nil, fmt.Errorf("tipo de constante desconhecido no bytecode: %d", valType)
		}
	}

	// 3. Lê o restante do Chunk (Code e Lines)
	if err := readChunkData(file, le, funcObj.Chunk); err != nil {
		return nil, err
	}

	return funcObj, nil
}

// readChunkData desserializa as seções Code e Lines de um chunk
func readChunkData(file io.Reader, le binary.ByteOrder, chunk *vm.Chunk) error { // <--- CORRETO
	var codeLen int64
	if err := binary.Read(file, le, &codeLen); err != nil {
		return err // <--- CORRETO
	}
	chunk.Code = make([]byte, codeLen)
	if _, err := io.ReadFull(file, chunk.Code); err != nil {
		return err // <--- CORRETO
	}
	var lineLen int64
	if err := binary.Read(file, le, &lineLen); err != nil {
		return err // <--- CORRETO
	}
	if lineLen != codeLen {
		return fmt.Errorf("inconsistência no bytecode (código %d != linhas %d)", codeLen, lineLen) // <--- CORRETO
	}
	chunk.Lines = make([]int, lineLen)
	for i := int64(0); i < lineLen; i++ {
		var line int64
		if err := binary.Read(file, le, &line); err != nil {
			return err // <--- CORRETO
		}
		chunk.Lines[i] = int(line)
	}
	return nil // <--- CORRETO
}

// readString é um helper para ler strings (tamanho + dados)
func readString(file io.Reader, le binary.ByteOrder) (string, error) {
	var strLen int64
	if err := binary.Read(file, le, &strLen); err != nil {
		return "", err
	}
	buf := make([]byte, strLen)
	if _, err := io.ReadFull(file, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"ion-language/pkg/compiler" // <--- Importante
	"ion-language/pkg/lexer"
	"ion-language/pkg/parser"
	"ion-language/pkg/vm"
)

var MagicNumber = []byte{0x49, 0x4F, 0x4E, 0x43} // "IONC"

func main() {
	// 1. Processar Flags
	sourceFile := flag.String("s", "", "Arquivo fonte .ion (ex: source.ion)")
	outputFile := flag.String("o", "", "Arquivo de saída de bytecode (ex: source.ionc)")
	flag.Parse()

	// --- ATUALIZAÇÃO V13.8: Prompt de Uso ---
	// Se os flags obrigatórios não foram passados (ou nenhum foi passado)
	if *sourceFile == "" || *outputFile == "" {
		fmt.Println("LabsObjects (R) Ion Optimizing Compiler Versão 1.13")
		fmt.Println("Copyright (C) LabsObjects. Todos os direitos reservados.")
		fmt.Println()
		fmt.Println("uso:  ionc -s <arquivo.ion> -o <arquivo.ionc>")
		os.Exit(0) // Sair sem erro (comportamento padrão de 'ajuda')
	}
	// --- FIM DA ATUALIZAÇÃO ---

	// 2. Ler o Arquivo Fonte
	data, err := ioutil.ReadFile(*sourceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao ler arquivo %s: %s\n", *sourceFile, err)
		os.Exit(1)
	}

	// 3. Pipeline do Parser
	l := lexer.New(string(data))
	p := parser.New(l)
	program := p.ParseProgram()
	errors := p.Errors()
	if len(errors) > 0 {
		fmt.Printf("❌ %d Erros de Sintaxe encontrados em %s:\n", len(errors), *sourceFile)
		printErrors(errors)
		os.Exit(1)
	}

	// 4. Pipeline do Compilador
	mainFunction, compErrors := compiler.Compile(program)
	if len(compErrors) > 0 {
		fmt.Printf("❌ %d Erros de Compilação encontrados em %s:\n", len(compErrors), *sourceFile)
		printErrors(compErrors)
		os.Exit(1)
	}

	// 5. Escrever o Arquivo de Bytecode
	err = writeBytecode(*outputFile, mainFunction)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao escrever arquivo de bytecode %s: %s\n", *outputFile, err)
		os.Exit(1)
	}

	fmt.Printf("✅ Compilação concluída com sucesso! -> %s\n", *outputFile)
}

// writeBytecode agora é um wrapper para writeFunction
func writeBytecode(filePath string, mainFunc *vm.FunctionObject) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 1. Escreve o Magic Number
	if _, err = file.Write(MagicNumber); err != nil {
		return err
	}

	// 2. Serializa recursivamente a função 'main'
	le := binary.LittleEndian
	return writeFunction(file, le, mainFunc)
}

// --- NOVOS HELPERS V9 (SERIALIZAÇÃO) ---

// writeFunction serializa um FunctionObject (Nome, Aridade, Chunk)
func writeFunction(file io.Writer, le binary.ByteOrder, funcObj *vm.FunctionObject) error {
	// 1. Escreve Nome e Aridade
	if err := writeString(file, le, funcObj.Name); err != nil {
		return err
	}
	if err := binary.Write(file, le, int64(funcObj.Arity)); err != nil {
		return err
	}

	// 2. Escreve a Piscina de Constantes do Chunk
	chunk := funcObj.Chunk
	if err := binary.Write(file, le, int64(len(chunk.Constants))); err != nil {
		return err
	}

	for _, val := range chunk.Constants {
		// Escreve o TIPO da constante
		if err := binary.Write(file, le, int8(val.Type)); err != nil {
			return err
		}
		// Escreve o DADO
		switch val.Type {
		case vm.VAL_NUMBER:
			if err := binary.Write(file, le, vm.AsNumber(val)); err != nil {
				return err
			}
		case vm.VAL_STRING:
			if err := writeString(file, le, vm.AsString(val)); err != nil {
				return err
			}
		case vm.VAL_FUNCTION:
			// É uma função aninhada! Serializa recursivamente.
			nestedFunc := vm.AsFunction(val)
			if err := writeFunction(file, le, nestedFunc); err != nil {
				return err
			}
		}
	}

	// 3. Escreve o restante do Chunk (Code e Lines)
	return writeChunkData(file, le, chunk)
}

// writeChunkData serializa as seções Code e Lines de um chunk
func writeChunkData(file io.Writer, le binary.ByteOrder, chunk *vm.Chunk) error {
	// 3a. Escreve o tamanho da seção (int64)
	if err := binary.Write(file, le, int64(len(chunk.Code))); err != nil {
		return err
	}
	// 3b. Escreve os bytes do código
	if _, err := file.Write(chunk.Code); err != nil {
		return err
	}
	// 4a. Escreve o número de entradas (int64)
	if err := binary.Write(file, le, int64(len(chunk.Lines))); err != nil {
		return err
	}
	// 4b. Escreve cada entrada de linha (int64)
	for _, line := range chunk.Lines {
		if err := binary.Write(file, le, int64(line)); err != nil {
			return err
		}
	}
	return nil
}

// writeString é um helper para escrever strings (tamanho + dados)
func writeString(file io.Writer, le binary.ByteOrder, s string) error {
	if err := binary.Write(file, le, int64(len(s))); err != nil {
		return err
	}
	if _, err := io.WriteString(file, s); err != nil {
		return err
	}
	return nil
}

// printErrors (Sem mudanças)
func printErrors(errors []string) {
	for _, msg := range errors {
		fmt.Printf("\t- %s\n", msg)
	}
}

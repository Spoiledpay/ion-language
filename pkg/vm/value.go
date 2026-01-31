package vm

import (
	"fmt"
	"strconv"
)

type FunctionObject struct {
	Arity int    // Quantos parâmetros a função espera
	Chunk *Chunk // O bytecode compilado da função
	Name  string // O nome da função (para depuração)
}

type ArrayObject struct {
	Values []Value // O contêiner de valores
}

// ValueType define o tipo de dado que estamos armazenando.
type ValueType int

const (
	VAL_NUMBER ValueType = iota
	VAL_STRING
	VAL_BOOL
	VAL_NIL
	VAL_FUNCTION
	VAL_ARRAY
)

// Value é a representação de qualquer dado dentro da VM Ion.
type Value struct {
	Type ValueType
	As   interface{} // Armazena float64, string ou bool
}

// --- Funções de "Criação" (Construtores) ---

func NewNumberValue(val float64) Value {
	return Value{Type: VAL_NUMBER, As: val}
}

func NewStringValue(val string) Value {
	return Value{Type: VAL_STRING, As: val}
}

func NewBoolValue(val bool) Value { // <--- ADICIONE ESTA FUNÇÃO
	return Value{Type: VAL_BOOL, As: val}
}

func NewNilValue() Value { // <--- ADICIONE ESTA FUNÇÃO
	return Value{Type: VAL_NIL, As: nil}
}

func NewFunctionValue(obj *FunctionObject) Value { // <--- ADICIONE ESTA FUNÇÃO
	return Value{Type: VAL_FUNCTION, As: obj}
}

// --- Funções de "Verificação" (Type Checking) ---

func IsNumber(val Value) bool {
	return val.Type == VAL_NUMBER
}

func IsString(val Value) bool {
	return val.Type == VAL_STRING
}

func IsBool(val Value) bool { // <--- ADICIONE ESTA FUNÇÃO
	return val.Type == VAL_BOOL
}

func IsNil(val Value) bool { // <--- ADICIONE ESTA FUNÇÃO
	return val.Type == VAL_NIL
}

func IsFunction(val Value) bool { // <--- ADICIONE ESTA FUNÇÃO
	return val.Type == VAL_FUNCTION
}

func IsArray(val Value) bool { // <--- ADICIONE
	return val.Type == VAL_ARRAY
}

// --- Funções de "Conversão" (Type Casting) ---

func AsNumber(val Value) float64 {
	return val.As.(float64)
}

func AsString(val Value) string {
	return val.As.(string)
}

func AsBool(val Value) bool { // <--- ADICIONE ESTA FUNÇÃO
	return val.As.(bool)
}

func AsFunction(val Value) *FunctionObject { // <--- ADICIONE ESTA FUNÇÃO
	return val.As.(*FunctionObject)
}

func AsArray(val Value) *ArrayObject { // <--- ADICIONE
	return val.As.(*ArrayObject)
}

func NewArrayValue(obj *ArrayObject) Value { // <--- ADICIONE
	return Value{Type: VAL_ARRAY, As: obj}
}

// PrintValue é como o 'display' do Ion vai imprimir os valores.
func PrintValue(val Value) {
	switch val.Type {
	case VAL_NUMBER:
		fmt.Printf("%g", AsNumber(val))
	case VAL_STRING:
		fmt.Printf("%s", AsString(val))
	case VAL_BOOL:
		fmt.Printf("%t", AsBool(val))
	case VAL_NIL:
		fmt.Printf("nil")
	case VAL_FUNCTION:
		fmt.Printf("<fn %s>", AsFunction(val).Name)
	case VAL_ARRAY:
		fmt.Printf("<array %d items>", len(AsArray(val).Values))
	}
}

// Stringify converte um valor para sua representação em string (para OP_CONCAT)
func Stringify(val Value) string {
	switch val.Type {
	case VAL_NUMBER:
		return strconv.FormatFloat(AsNumber(val), 'g', -1, 64)
	case VAL_STRING:
		return AsString(val)
	case VAL_BOOL:
		return strconv.FormatBool(AsBool(val))
	case VAL_NIL:
		return "nil"
	case VAL_FUNCTION:
		return fmt.Sprintf("<fn %s>", AsFunction(val).Name)
	case VAL_ARRAY:
		return fmt.Sprintf("<array %d items>", len(AsArray(val).Values))
	default:
		return "desconhecido"
	}
}

package tuplespace

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

const (
	// FLOATPRECISION sets the error for floating point comparison
	FLOATPRECISION float64 = 0.0000001
)

type TupleElement uint8

const (
	// INT indicates 32bit-integers.
	INT TupleElement = 1
	// FLOAT indicates double precision (64bit) floating point numbers.
	FLOAT = 2
	// STRING indicates... well... strings.
	STRING = 3
	// TUPLE indicates a nested tuple.
	TUPLE = 4
	// ANY indicates any possible type of the above, functioning as a wildcard.
	ANY = 5
	// NONE indicates an invalid type
	NONE = 0
)

const (
	// LT is the `less than` return value for order comparisons
	LT int = -1
	// EQ is the `equals` return value for order comparisons
	EQ int = 0
	// GT is the `greater than` return value for order comparisons
	GT int = 1
)

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// Elem acts as an element container, holding a generic element and its type indication.
type Elem struct {
	elemType  TupleElement
	elemValue interface{}
}

func (e Elem) GetType() TupleElement {
	return e.elemType
}

func (e Elem) GetValue() interface{} {
	return e.elemValue
}

func (e Elem) String() string {
	switch e.elemType {
	case INT:
		return fmt.Sprintf("%v", e.elemValue.(int))
	case FLOAT:
		return fmt.Sprintf("%v", e.elemValue.(float64))
	case STRING:
		return fmt.Sprintf("\"%v\"", e.elemValue.(string))
	case TUPLE:
		return e.elemValue.(Tuple).String()
	case ANY:
		return "_"
	case NONE:
		return "nil"
	default:
		panic(fmt.Sprintf("Error: invalid elem type %T", e.elemValue))
	}
}

// Tuple element constructors /////////////////////////////////////////////////

// I instantiates an int-type tuple element.
func I(intVal int) Elem {
	return Elem{INT, intVal}
}

// F instantiates a double precision (64bit) float64-type tuple element.
func F(floatVal float64) Elem {
	return Elem{FLOAT, floatVal}
}

// S instantiates a string-type tuple element.
func S(stringVal string) Elem {
	return Elem{STRING, stringVal}
}

// T instantiates a Tuple-type tuple element.
func T(tupleVal Tuple) Elem {
	return Elem{TUPLE, tupleVal}
}

// A instantiates a Wildcard tuple element.
func Any() Elem {
	return Elem{ANY, nil}
}

func None() Elem {
	return Elem{NONE, nil}
}

// Returns true if the element is defined, false if it is a wildcard or none
func (e Elem) IsDefined() bool {
	switch e.elemType {
	case INT:
		return true
	case FLOAT:
		return true
	case STRING:
		return true
	case TUPLE:
		return e.elemValue.(Tuple).IsDefined()
	case ANY:
		return false
	case NONE:
		return false
	default:
		panic(fmt.Sprintf("Error: invalid elem type %T", e.elemValue))
	}
}

// Match two elements for equality, which is true either if they are of the same type and value
// or one or both are wildcards.
func (e Elem) isMatching(other Elem) bool {
	if e.elemType == INT && other.elemType == INT {
		return e.elemValue.(int) == other.elemValue.(int)
	}

	if e.elemType == FLOAT && other.elemType == FLOAT {
		return (math.Abs(e.elemValue.(float64)-other.elemValue.(float64)) < FLOATPRECISION)
	}

	if e.elemType == STRING && other.elemType == STRING {
		return e.elemValue.(string) == other.elemValue.(string)
	}
	if e.elemType == TUPLE && other.elemType == TUPLE {
		return e.elemValue.(Tuple).IsMatching(other.elemValue.(Tuple))
	}

	if e.elemType == NONE || other.elemType == NONE {
		return false
	}

	if e.elemType == ANY || other.elemType == ANY {
		return true
	}
	return false
}

// Comparator function, used for determining ordering of two elements.
// The order between elements of different type is arbitrary, but consistent.
// ANY < tuple < string < double < int < nil
// The order between elements of the same type is the builtin in golang
// Note to self: discussion about value receiver vs pointer receiver:
//
//	https://stackoverflow.com/questions/27775376/value-receiver-vs-pointer-receiver-in-golang
//
// Returns 1 if this e > other, -1 if e < other, 0 if both are equal
func (e Elem) order(other Elem) int {
	switch e.elemType {

	case ANY:
		return EQ

	case TUPLE:
		switch other.elemType {
		case ANY:
			return EQ
		case TUPLE:
			return e.elemValue.(Tuple).order(other.elemValue.(Tuple))
		default:
			return LT
		}

	case STRING:
		switch other.elemType {
		case ANY:
			return EQ
		case TUPLE:
			return GT
		case STRING:
			if e.elemValue.(string) < other.elemValue.(string) {
				return LT
			}
			if e.elemValue.(string) == other.elemValue.(string) {
				return EQ
			}
			return GT
		case FLOAT:
		case INT:
		case NONE:
			return LT
		}

	case FLOAT:
		switch other.elemType {
		case ANY:
			return EQ
		case TUPLE:
		case STRING:
			return GT
		case FLOAT:
			if e.elemValue.(float64) < other.elemValue.(float64) {
				return LT
			}
			if e.elemValue.(float64) == other.elemValue.(float64) {
				return EQ
			}
			return GT
		case INT:
		case NONE:
			return LT
		}

	case INT:
		if other.elemType == ANY {
			return EQ
		}
		if other.elemType == INT {
			if e.elemValue.(int) < other.elemValue.(int) {
				return LT
			}
			if e.elemValue.(int) == other.elemValue.(int) {
				return EQ
			}
			return GT
		}
		if other.elemType == NONE {
			return GT
		}
		return LT
	default:
		return LT
	}
	return LT
}

// Tuple can contain elements of five different data types:
// - integers
// - floating point numbers
// - strings
// - tuples themselves
// - wildcards
type Tuple struct {
	elements []Elem
}

func (t Tuple) String() string {
	var strBuilder strings.Builder
	strBuilder.WriteString("(")

	size := len(t.elements)
	for i, e := range t.elements {
		strBuilder.WriteString(e.String())
		if i < size-1 {
			strBuilder.WriteString("|")
		}
	}
	strBuilder.WriteString(")")
	return strBuilder.String()
}

// MakeTuple creates a new Tuple instance from the given parameters.
func MakeTuple(element ...Elem) Tuple {
	var resultTuple Tuple
	// For debugging only.
	// for _, e := range element {
	// 	fmt.Printf("element %v of type %T (reflect %v)\n", e, e, reflect.TypeOf(e))
	// }
	resultTuple.elements = element
	// fmt.Printf("resulting tuple: %v", resultTuple)
	return resultTuple
}

// Returns true if the tuple does not contain any wildcards or none fields
func (t Tuple) IsDefined() bool {
	for _, v := range t.elements {
		if !v.IsDefined() {
			// break out of the loop if any element is undefined
			return false
		}
	}
	return true
}

// IsMatching checks two tuples for equality, which is true if
// - they are of the same lenght AND
// - each element of one matches the others
func (t Tuple) IsMatching(other Tuple) bool {
	tSize := len(t.elements)
	otherSize := len(other.elements)

	// Check length of tuples first
	if tSize != otherSize {
		return false
	}

	// Check each element for equality.
	for i := 0; i < tSize; i++ {
		if !t.elements[i].isMatching(other.elements[i]) {
			return false
		}
	}

	return true
}

// Comparator function, used for determining ordering of two tuples.
func (t Tuple) order(other Tuple) int {
	tSize := len(t.elements)
	otherSize := len(other.elements)

	shorterSize := min(tSize, otherSize)

	// Check each element for equality.
	for i := 0; i < shorterSize; i++ {
		if ord := t.elements[i].order(other.elements[i]); ord != EQ {
			return ord
		}
	}

	if tSize < otherSize {
		return LT
	}

	if tSize == otherSize {
		return EQ
	}

	return GT
}

// Comparator function for ordering tuples, used for tidwall/btree.
// Returns `true` if t1 is considered _less than_ t2, `false` otherwise.
func TupleOrder(t1, t2 Tuple) bool {
	return t1.order(t2) == LT
}

func float64frombytes(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint64(bytes)
	float := math.Float64frombits(bits)
	return float
}

func float64bytes(float float64) []byte {
	bits := math.Float64bits(float)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, bits)
	return bytes
}

// Encode type and value of each element into a byte slice.
// Doesn't support nested tuples yet.
func EncodeTuple(t Tuple) []byte {
	var result []byte
	for _, e := range t.elements {
		switch e.GetType() {
		case INT:
			var intVal int = e.GetValue().(int)
			var varType byte = byte(INT)
			var toBytes []byte = make([]byte, 4)
			binary.LittleEndian.PutUint32(toBytes, uint32(intVal))
			var bytes []byte = append([]byte{varType}, toBytes...)
			result = append(result, bytes...)
		case FLOAT:
			var floatVal float64 = e.GetValue().(float64)
			var varType byte = byte(FLOAT)
			toBytes := float64bytes(floatVal)
			var bytes []byte = append([]byte{varType}, toBytes...)
			result = append(result, bytes...)
		case STRING:
			var str string = e.GetValue().(string)
			var len int = len(str)
			var varType byte = byte(STRING)
			var bytes []byte = append([]byte{varType, byte(len)}, []byte(str)...)
			result = append(result, bytes...)
		case ANY:
			result = append(result, byte(ANY))
		case NONE:
			result = append(result, byte(NONE))
		}
	}

	return result
}

// Decode a byte slice into a tuple.
func DecodeTuple(data []byte) Tuple {
	var resultTuple Tuple
	var elemList []Elem
	var elemType TupleElement
	var elemLen int
	var elemValue interface{}

	fmt.Printf("Len of data: %v\n", len(data))
	for i := 0; i < len(data); {
		elemType = TupleElement(data[i])
		switch elemType {
		case INT:
			elemLen = 4
			elemValue = int(binary.LittleEndian.Uint32(data[i+1 : i+1+elemLen]))
			fmt.Printf("decoded int: %v with len %v\n", elemValue, elemLen)
			elemList = append(elemList, Elem{elemType, elemValue})
			i += elemLen + 1
		case FLOAT:
			elemLen := 8
			elemValue := float64frombytes(data[i+1 : i+1+elemLen])
			elemList = append(elemList, Elem{elemType, elemValue})
			i += elemLen + 1
		case STRING:
			elemLen = int(data[i+1])
			elemValue = string(data[i+2 : i+2+elemLen])
			elemList = append(elemList, Elem{elemType, elemValue})
			i += elemLen + 2
		case ANY:
			elemList = append(elemList, Elem{elemType, nil})
			i++
		case NONE:
			elemList = append(elemList, Elem{elemType, nil})
			i++
		}
	}
	resultTuple.elements = elemList
	return resultTuple
}

package main

import (
	"fmt"
	parser "tuplespaceCD/cmd/server"
	ts "tuplespaceCD/pkg/tuplespace"
)

func main() {

	// Read a string from the command line
	var input string
	fmt.Print("Enter a tuple: ")
	fmt.Scanln(&input)

	// Parse the string as a tuple
	lexer := parser.NewLexer(input)
	tuples, err := lexer.IntoTuples()

	if err != nil {
		fmt.Println(err)
		return
	}

	// Encode and decode the tuple
	for _, tuple := range tuples {
		fmt.Printf("Parsed tuple: %v\n", tuple)
		encoded := ts.EncodeTuple(tuple)
		decoded := ts.DecodeTuple(encoded)

		// Print the original tuple
		fmt.Printf("Original tuple: %v\n", tuple)

		// Print the encoded tuple
		fmt.Printf("Encoded tuple: %v\n", encoded)

		// Print the decoded tuple
		fmt.Printf("Decoded tuple: %v\n", decoded)
	}

}

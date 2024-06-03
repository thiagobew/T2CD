package main

import (
	"fmt"
	parser "tuplespaceCD/cmd/server"
	ts "tuplespaceCD/pkg/tuplespace"
)

func main() {
	space := ts.NewSpace()

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

	for _, tuple := range tuples {
		// Insert the tuple into the tuple space
		<-space.Out(tuple)
	}

	for _, tuple := range tuples {
		// Retrieve the tuple from the tuple space
		recv := <-space.In(tuple)
		// Print the tuple
		fmt.Println(recv)
	}

}

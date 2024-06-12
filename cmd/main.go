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
	fmt.Printf("Parsing tuple: %s\n", input)
	lexer := parser.NewLexer(input)
	tuples, err := lexer.IntoTuples()

	if err != nil {
		fmt.Println(err)
		return
	}

	space := ts.NewSpace()

	// Write the tuple to the tuple space
	for _, tuple := range tuples {
		<-space.Write(tuple)
	}

	// Read the tuple from the tuple space
	for _, tuple := range tuples {
		fmt.Println("Reading tuple: ", tuple)
		result := <-space.Read(tuple)
		if result.IsPresent() {
			fmt.Println("Result: ", result.Get())
		} else {
			fmt.Println("No result found")
		}
	}

}

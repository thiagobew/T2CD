package main

import (
	"fmt"
	"net"
	ts "tuplespaceCD/pkg/tuplespace"
)

func handleRequest(addr net.Addr, basePort uint16, space *ts.Space) {
	// Open new connection to the client
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", addr.String(), basePort))
	if err != nil {
		fmt.Println("Error connecting to client:", addr.String(), basePort)
		conn.Close()
		return
	}
	defer conn.Close()

	// TODO: Search for a REQ tuple with the given id

	// Write the response back to the client
	_, err = conn.Write([]byte("Response to request"))
	if err != nil {
		fmt.Println("Error sending response:", err)
	}
}

func server() {
	// Start the server
	space := ts.NewSpace()

	conn, err := net.Listen("tcp", ":1235")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Server started on port 1235")

	var basePort uint16 = 1236

	buffer := make([]byte, 1024)

	for {
		conn, err := conn.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		defer conn.Close()

		fmt.Println("Received request from", conn.RemoteAddr().String())

		_, err = conn.Read(buffer)
		if err != nil {
			fmt.Println("Error reading request:", err)
			conn.Close()
			continue
		}

		// Write base port to the client
		_, err = conn.Write([]byte(fmt.Sprintf("%d", basePort)))
		if err != nil {
			fmt.Println("Error sending base port:", err)
		}

		// TODO: parse bytes into tuple request, writing to space

		handleRequest(conn.RemoteAddr(), basePort, space)
		basePort += 1
	}
}

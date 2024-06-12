// server
package main

import (
	"encoding/json"
	"fmt"
	"net"
	ts "tuplespaceCD/pkg/tuplespace"

	"github.com/micutio/goptional"
)

type Request struct {
	BankAccount     string
	Password        string
	Requisition     string
	RequisitionData string
}

func handleClient(space *ts.Space, basePort uint16) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", basePort))
	if err != nil {
		fmt.Println("Error starting listener on new port:", err)
		return
	}
	defer listener.Close()

	conn, err := listener.Accept()
	if err != nil {
		fmt.Println("Error accepting connection on new port:", err)
		return
	}
	defer conn.Close()

	var req Request
	err = json.NewDecoder(conn).Decode(&req)
	if err != nil {
		fmt.Println("Error decoding request:", err)
		return
	}

	space.Write(ts.MakeTuple(ts.S("REQ"), ts.S(req.BankAccount), ts.S(req.Password), ts.S(req.Requisition), ts.S(req.RequisitionData)))

	var resp goptional.Maybe[ts.Tuple]
	for {
		respChan := space.Get(ts.MakeTuple(ts.S("RESP"), ts.S(req.BankAccount), ts.Any()))
		resp = <-respChan

		if resp.IsPresent() {
			break
		}
	}

	responseData, err := json.Marshal(resp)
	if err != nil {
		fmt.Println("Error encoding response:", err)
		return
	}

	_, err = conn.Write(responseData)
	if err != nil {
		fmt.Println("Error sending response:", err)
	}
}

func main() {
	space := ts.NewSpace()
	listener, err := net.Listen("tcp", ":1235")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server started on port 1235")

	var basePort uint16 = 1236

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		fmt.Println("Received connection from", conn.RemoteAddr())

		// Send the new port to the client
		_, err = conn.Write([]byte(fmt.Sprintf("%d", basePort)))
		if err != nil {
			fmt.Println("Error sending new port to client:", err)
			conn.Close()
			continue
		}
		conn.Close()

		go handleClient(space, basePort)
		basePort++
	}
}

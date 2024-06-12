// client
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Request struct {
	BankAccount     string
	Password        string
	Requisition     string
	RequisitionData string
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter command: ")
		cmd, _ := reader.ReadString('\n')
		cmd = strings.TrimSpace(cmd)
		args := strings.Split(cmd, " ")

		if len(args) < 3 {
			fmt.Println("Invalid command")
			continue
		}

		bankAccount := args[0]
		password := args[1]
		requisition := args[2]
		requisitionData := ""
		if len(args) > 3 {
			requisitionData = strings.Join(args[3:], " ")
		}

		req := Request{
			BankAccount:     bankAccount,
			Password:        password,
			Requisition:     requisition,
			RequisitionData: requisitionData,
		}

		// Connect to the server on the initial port
		conn, err := net.Dial("tcp", "localhost:1235")
		if err != nil {
			fmt.Println("Error connecting to server:", err)
			continue
		}

		// Read the new port from the server
		var portBuf [5]byte
		_, err = conn.Read(portBuf[:])
		if err != nil {
			fmt.Println("Error reading new port:", err)
			conn.Close()
			continue
		}
		conn.Close()

		newPort, err := strconv.Atoi(strings.TrimSpace(string(portBuf[:])))
		if err != nil {
			fmt.Println("Invalid port received:", err)
			continue
		}

		// Connect to the server on the new port
		newConn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", newPort))
		if err != nil {
			fmt.Println("Error connecting to server on new port:", err)
			continue
		}

		// Send the request
		err = json.NewEncoder(newConn).Encode(req)
		if err != nil {
			fmt.Println("Error sending request:", err)
			newConn.Close()
			continue
		}

		// Read the response
		var resp map[string]interface{}
		err = json.NewDecoder(newConn).Decode(&resp)
		if err != nil {
			fmt.Println("Error reading response:", err)
			newConn.Close()
			continue
		}

		newConn.Close()
		fmt.Println("Response:", resp)
	}
}

// Example usage of the client once everything is running:
// Enter command: 1234 pass create
// Enter command: 1234 pass deposit 100
// Enter command: 1234 pass withdraw 50
// Enter command: 1234 pass delete

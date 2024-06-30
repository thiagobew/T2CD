// client
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type Request struct {
	BankAccount     string
	Password        string
	Requisition     string
	RequisitionData string
}

type JSONConnectionInfo struct {
	MesType  string `json:"type"`
	NodeAddr string `json:"addr"`
	NodeID   string `json:"id"`
}

func printCommands() {
	fmt.Println("Commands:")
	fmt.Println("  <bankAccount> <password> create")
	fmt.Println("  <bankAccount> <password> deposit <amount>")
	fmt.Println("  <bankAccount> <password> withdraw <amount>")
	fmt.Println("  <bankAccount> <password> delete")
	fmt.Println("  <bankAccount> <password> balance")
}

// TODO: refactor connect logic into a function
func main() {
	reader := bufio.NewReader(os.Stdin)
	// TODO: get start port and address from command line
	var serverPort uint16 = 11000

	for {
		printCommands()
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
		var conn net.Conn
		var err error
		for {
			for i := 0; i < 5; i++ {
				conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", serverPort))
				if err == nil {
					break
				}
				fmt.Println("Error connecting to server:", err)
				time.Sleep(1 * time.Second)
			}
			if err == nil {
				break
			}

			serverPort += 100
		}

		// Write to the server to inform a request
		info := JSONConnectionInfo{
			MesType:  "request",
			NodeAddr: "",
			NodeID:   "",
		}
		b, err := json.Marshal(info)
		if err != nil {
			fmt.Println("Error encoding JSON:", err)
			conn.Close()
			continue
		}
		conn.Write(b)

		// Read the new port from the server
		// TODO: implement with JSON like { "port": uint16, "addr": string, "leader": bool }
		// If leader is true, then the client can send the request to the server
		// If leader is false, then the client must repeat the connection process with the leader
		// TODO: treat case when there's no leader (sleep with exponential backoff and retry?)
		var portBuf [2]byte
		_, err = conn.Read(portBuf[:])
		if err != nil {
			fmt.Println("Error reading new port:", err)
			conn.Close()
			continue
		}
		conn.Close()

		var newPort uint16 = uint16(portBuf[1])<<8 | uint16(portBuf[0])

		// Connect to the server on the new port
		var newConn net.Conn
		for {
			newConn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", newPort))
			fmt.Printf("Connecting to localhost:%d\n", newPort)
			if err == nil {
				break
			}
			time.Sleep(300 * time.Millisecond)
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

		// TODO: check if response didn't return an error
		// If it did, get leader address, connect to him, and repeat the request
		fmt.Println("Response:", resp)
	}
}

// Example usage of the client once everything is running:
// Enter command: 1234 pass create
// Enter command: 1234 pass deposit 100
// Enter command: 1234 pass withdraw 50
// Enter command: 1234 pass delete

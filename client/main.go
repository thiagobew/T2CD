// client
package main

import (
	"bufio"
	"encoding/json"
	"flag"
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

func exponentialBackoff(retries uint) time.Duration {
	return time.Duration(1<<retries) * time.Millisecond
}

func tryConnect(address string, port uint16, retries int) (net.Conn, error) {
	var conn net.Conn
	var err error
	for i := 0; i < retries; i++ {
		conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", address, port))
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	return conn, err
}

func findServer(startAddress string, startPort uint16, tries uint) (net.Conn, uint16, error) {
	port := startPort
	var conn net.Conn
	var err error
	for {
		conn, err = tryConnect(startAddress, port, 3)
		if err == nil {
			break
		}
		port += 100
	}

	// Write to the server to inform a request
	info := JSONConnectionInfo{
		MesType:  "request",
		NodeAddr: "",
		NodeID:   "",
	}
	b, err := json.Marshal(info)
	if err != nil {
		conn.Close()
		return nil, 0, err
	}
	conn.Write(b)

	var response map[string]interface{}

	err = json.NewDecoder(conn).Decode(&response)
	if err != nil {
		conn.Close()
		return nil, 0, err
	}

	addr := response["addr"].(string)
	isLeader := response["leader"].(bool)

	if !isLeader {
		conn.Close()

		if addr == "" {
			fmt.Printf("No leader found, timeout and try again\n")
			time.Sleep(exponentialBackoff(tries))
			return findServer(startAddress, startPort, tries+1)
		}

		fmt.Printf("Redirected to %s\n", addr)
		// Strip port from address
		addr = strings.Split(addr, ":")[0]

		portToAsk := startPort
		if addr == startAddress {
			portToAsk = portToAsk + 100
		}

		return findServer(addr, portToAsk, tries)
	}

	// Read the new port from the server
	var portBuf [2]byte
	_, err = conn.Read(portBuf[:])
	if err != nil {
		fmt.Println("Error reading new port:", err)
		conn.Close()
		return nil, 0, err
	}
	conn.Close()

	var newPort uint16 = uint16(portBuf[1])<<8 | uint16(portBuf[0])

	// Connect to the server on the new port
	var newConn net.Conn
	newConn, err = tryConnect(startAddress, newPort, 3)
	if err != nil {
		return nil, 0, err
	}

	return newConn, newPort, nil
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	var serverPort uint16
	// Get address and port from command line
	var address string
	var port uint

	flag.StringVar(&address, "address", "localhost", "Server address")
	flag.UintVar(&port, "port", 11000, "Server port")
	flag.Parse()

	serverPort = uint16(port)

	// Connect to the server
	conn, newPort, err := findServer(address, serverPort, 0)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}

	fmt.Printf("Connected to %s:%d\n", address, newPort)

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

		// Send the request
		err = json.NewEncoder(conn).Encode(req)
		if err != nil {
			fmt.Println("Error sending request:", err)
			conn.Close()
			continue
		}

		// Read the response
		var resp map[string]interface{}
		err = json.NewDecoder(conn).Decode(&resp)
		if err != nil {
			conn.Close()
			if err.Error() == "EOF" {
				fmt.Println("Reconnecting to server")
				conn, newPort, err = findServer(address, serverPort, 0)
				if err != nil {
					fmt.Println("Error reconnecting to server:", err)
					return
				}
				fmt.Printf("Connected to %s:%d. Repeat the request, please.\n", address, newPort)

			}

			continue
		}

		fmt.Println("Response:")
		for k, v := range resp {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}
}

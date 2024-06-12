// server
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"
	ts "tuplespaceCD/pkg/tuplespace"

	"github.com/micutio/goptional"
)

type Request struct {
	BankAccount     string
	Password        string
	Requisition     string
	RequisitionData string
}

func worker(space *ts.Space) {
	for {
		query := ts.MakeTuple(ts.S("REQ"), ts.Any(), ts.Any(), ts.Any(), ts.Any())
		reqChan := space.Get(query)
		req := <-reqChan // Read from the channel
		if req.IsPresent() {
			fmt.Printf("Worker got req: %v\n", req)
			bankAccount := req.Get().GetElements()[1].String()
			password := req.Get().GetElements()[2].String()
			requisition := req.Get().GetElements()[3].String()
			requisitionData := req.Get().GetElements()[4].String()

			fmt.Printf("Processing request: %s %s %s %s\n", bankAccount, password, requisition, requisitionData)

			switch requisition {
			case `"create"`:
				var wrote bool

				wrote = <-space.Write(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.S(requisitionData)))
				fmt.Printf("Wrote account: %v\n", wrote)
				wrote = <-space.Write(ts.MakeTuple(ts.S("RES"), ts.S(bankAccount), ts.S("Account created")))
				fmt.Printf("Wrote response: %v\n", wrote)

			case `"delete"`:
				<-space.Get(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.Any()))
				<-space.Write(ts.MakeTuple(ts.S("RES"), ts.S(bankAccount), ts.S("Account deleted")))

			case `"deposit"`:
				tupleChan := space.Get(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.Any()))
				tuple := <-tupleChan
				if tuple != nil {
					moneyStr := tuple.Get().GetElements()[2].String()
					money, _ := strconv.Atoi(moneyStr)
					depositAmount, _ := strconv.Atoi(requisitionData)
					<-space.Write(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.I(money+depositAmount)))
					<-space.Write(ts.MakeTuple(ts.S("RES"), ts.S(bankAccount), ts.S("Deposit successful")))
				} else {
					<-space.Write(ts.MakeTuple(ts.S("RES"), ts.S(bankAccount), ts.S("Account not found")))
				}

			case `"withdraw"`:
				tupleChan := space.Get(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.Any()))
				tuple := <-tupleChan
				if tuple != nil {
					moneyStr := tuple.Get().GetElements()[2].String()
					money, _ := strconv.Atoi(moneyStr)
					withdrawAmount, _ := strconv.Atoi(requisitionData)
					if money >= withdrawAmount {
						<-space.Write(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.I(money-withdrawAmount)))
						<-space.Write(ts.MakeTuple(ts.S("RES"), ts.S(bankAccount), ts.S("Withdrawal successful")))
					} else {
						<-space.Write(ts.MakeTuple(ts.S("RES"), ts.S(bankAccount), ts.S("Insufficient funds")))
					}
				} else {
					<-space.Write(ts.MakeTuple(ts.S("RES"), ts.S(bankAccount), ts.S("Account not found")))
				}
			}
		}
		time.Sleep(1 * time.Second) // Sleep for a second
	}
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

	fmt.Printf("Received request: %v\n", req)

	// Write the request to the tuple space
	tuple := ts.MakeTuple(ts.S("REQ"), ts.S(req.BankAccount), ts.S(req.Password), ts.S(req.Requisition), ts.S(req.RequisitionData))
	fmt.Printf("Writing tuple: %v\n", tuple)

	<-space.Write(tuple)

	var resp goptional.Maybe[ts.Tuple]
	for {
		respChan := space.Get(ts.MakeTuple(ts.S("RES"), ts.S(req.BankAccount), ts.Any()))
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
	go worker(space)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleClient(space, basePort)

		fmt.Println("Received connection from", conn.RemoteAddr())

		// Send the new port to the client
		_, err = conn.Write([]byte(fmt.Sprintf("%d", basePort)))
		if err != nil {
			fmt.Println("Error sending new port to client:", err)
			conn.Close()
			continue
		}
		conn.Close()

		basePort++
	}
}

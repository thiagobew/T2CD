package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	ts "tuplespaceCD/pkg/tuplespace"
	"tuplespaceCD/store"

	"github.com/micutio/goptional"
)

// Command line defaults
const (
	DefaultHTTPAddr = "localhost:11000"
	DefaultRaftAddr = "localhost:12000"
)

// Command line parameters
var httpAddr string // server address
var raftAddr string
var joinAddr string
var nodeID string

func init() {
	flag.StringVar(&httpAddr, "haddr", DefaultHTTPAddr, "Set the HTTP bind address")
	flag.StringVar(&raftAddr, "raddr", DefaultRaftAddr, "Set Raft bind address")
	flag.StringVar(&joinAddr, "join", "", "Set join address, if any")
	flag.StringVar(&nodeID, "id", "", "Node ID. If not set, same as Raft bind address")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <raft-data-path> \n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "No Raft storage directory specified\n")
		os.Exit(1)
	}

	if nodeID == "" {
		nodeID = raftAddr
	}

	// Ensure Raft storage exists.
	raftDir := flag.Arg(0)
	if raftDir == "" {
		log.Fatalln("No Raft storage directory specified")
	}
	if err := os.MkdirAll(raftDir, 0700); err != nil {
		log.Fatalf("failed to create path for Raft storage: %s", err.Error())
	}

	s := store.New()
	s.RaftDir = raftDir
	s.RaftBind = raftAddr
	if err := s.Open(joinAddr == "", nodeID); err != nil {
		log.Fatalf("failed to open store: %s", err.Error())
	}

	// If join was specified, make the join request.
	if joinAddr != "" {
		if err := join(joinAddr, raftAddr, nodeID); err != nil {
			log.Fatalf("failed to join node at %s: %s", joinAddr, err.Error())
		}
	}

	// We're up and running!
	log.Printf("hraftd started successfully")

	go startServer(s, httpAddr)

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	log.Println("hraftd exiting")
}

func join(joinAddr, raftAddr, nodeID string) error {
	info := JSONConnectionInfo{
		MesType:  "join",
		NodeAddr: raftAddr,
		NodeID:   nodeID,
	}

	b, err := json.Marshal(info)
	if err != nil {
		return err
	}

	conn, err := net.Dial("tcp", joinAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(b)
	if err != nil {
		return err
	}

	return nil
}

type basePortControl struct {
	basePort uint16
	mutex    sync.Mutex
}

type JSONConnectionInfo struct {
	MesType  string `json:"type"`
	NodeAddr string `json:"addr"`
	NodeID   string `json:"id"`
}

func startServer(space *store.Store, address string) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server started on %s", address)

	var basePort uint16
	parts := strings.Split(address, ":")
	if len(parts) > 1 {
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			fmt.Println("Error parsing port:", err)
			return
		}
		basePort = uint16(port) + 1
	} else {
		fmt.Println("Invalid address format")
		return
	}

	basePortControl := &basePortControl{basePort: basePort}

	go worker(space)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleClient(space, basePortControl, basePortControl.basePort)

		fmt.Println("Received connection from", conn.RemoteAddr())

		// Decide if it's a request or a join
		var buf []byte
		_, err = conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading from connection:", err)
			conn.Close()
			continue
		}

		var info JSONConnectionInfo
		err = json.NewDecoder(conn).Decode(&info)
		if err != nil {
			fmt.Println("Error decoding JSON:", err)
			conn.Close()
			continue
		}
		fmt.Printf("Received JSON: %v\n", info)

		if info.MesType == "join" {
			space.Join(info.NodeID, info.NodeAddr)
			conn.Close()
			continue
		}

		// Send the new port to the client
		fmt.Printf("Sending new port to client: %d\n", basePort)
		_, err = conn.Write([]byte{byte(basePort), byte(basePort >> 8)})
		if err != nil {
			fmt.Println("Error sending new port to client:", err)
			conn.Close()
			continue
		}
		conn.Close()

		basePortControl.mutex.Lock()
		basePortControl.basePort++
		basePortControl.mutex.Unlock()
	}
}

type Request struct {
	BankAccount     string
	Password        string
	Requisition     string
	RequisitionData string
}

type Response struct {
	BankAccount string
	Message     string
}

func worker(space *store.Store) {
	for {
		query := ts.MakeTuple(ts.S("REQ"), ts.Any(), ts.Any(), ts.Any(), ts.Any())
		req, err := space.Get(query)
		if err != nil {
			//fmt.Println("Error getting request:", err)
			continue
		}

		if req.IsPresent() {
			fmt.Printf("Worker got req: %v\n", req)
			bankAccount := strings.Trim(req.Get().GetElements()[1].String(), `"`)
			password := strings.Trim(req.Get().GetElements()[2].String(), `"`)
			requisition := strings.Trim(req.Get().GetElements()[3].String(), `"`)
			requisitionData := strings.Trim(req.Get().GetElements()[4].String(), `"`)

			fmt.Printf("Processing request: %s %s %s %s\n", bankAccount, password, requisition, requisitionData)

			switch requisition {
			case "create":
				var err error

				err = space.Write(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.S(requisitionData)))
				fmt.Printf("Wrote account. Error: %v\n", err)
				err = space.Write(ts.MakeTuple(ts.S("RES"), ts.S(bankAccount), ts.S("Account created")))
				fmt.Printf("Wrote response, Error: %v\n", err)

			case "delete":
				space.Get(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.Any()))
				space.Write(ts.MakeTuple(ts.S("RES"), ts.S(bankAccount), ts.S("Account deleted")))

			case "deposit":
				tuple, err := space.Get(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.Any()))
				if err != nil {
					fmt.Println("Error getting tuple:", err)
					continue
				}

				if tuple.IsPresent() {
					moneyStr := tuple.Get().GetElements()[2].String()
					money, _ := strconv.Atoi(moneyStr)
					depositAmount, _ := strconv.Atoi(requisitionData)
					space.Write(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.I(money+depositAmount)))
					space.Write(ts.MakeTuple(ts.S("RES"), ts.S(bankAccount), ts.S("Deposit successful")))
				} else {
					space.Write(ts.MakeTuple(ts.S("RES"), ts.S(bankAccount), ts.S("Account not found")))
				}

			case "withdraw":
				tuple, err := space.Get(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.Any()))
				if err != nil {
					fmt.Println("Error getting tuple:", err)
					continue
				}

				if tuple.IsPresent() {
					moneyStr := tuple.Get().GetElements()[2].String()
					money, _ := strconv.Atoi(moneyStr)
					withdrawAmount, _ := strconv.Atoi(requisitionData)
					if money >= withdrawAmount {
						space.Write(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.I(money-withdrawAmount)))
						space.Write(ts.MakeTuple(ts.S("RES"), ts.S(bankAccount), ts.S("Withdrawal successful")))
					} else {
						space.Write(ts.MakeTuple(ts.S("RES"), ts.S(bankAccount), ts.S("Insufficient funds")))
					}
				} else {
					space.Write(ts.MakeTuple(ts.S("RES"), ts.S(bankAccount), ts.S("Account not found")))
				}
			case "balance":
				tuple, err := space.Read(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.Any()))
				if err != nil {
					fmt.Println("Error getting tuple:", err)
					continue
				}

				if tuple.IsPresent() {
					moneyStr := tuple.Get().GetElements()[2].String()
					space.Write(ts.MakeTuple(ts.S("RES"), ts.S(bankAccount), ts.S("Balance: "+moneyStr)))
				} else {
					space.Write(ts.MakeTuple(ts.S("RES"), ts.S(bankAccount), ts.S("Account not found")))
				}
			}
		}
		time.Sleep(1 * time.Second) // Sleep for a second
	}
}

func handleClient(space *store.Store, basePortCtl *basePortControl, basePort uint16) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", basePort))
	fmt.Printf("Listening on port: %d\n", basePort)
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

	defer func() {
		basePortCtl.mutex.Lock()
		basePortCtl.basePort--
		basePortCtl.mutex.Unlock()
	}()

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

	// TODO: If error, return leader address to client
	space.Write(tuple)

	var resp goptional.Maybe[ts.Tuple]
	var respData Response

	for {
		resp, err := space.Get(ts.MakeTuple(ts.S("RES"), ts.S(req.BankAccount), ts.Any()))
		if err != nil {
			fmt.Println("Error getting response:", err)
		}

		if resp.IsPresent() {
			fmt.Printf("Got response: %s\n", resp)
			respData = Response{
				BankAccount: resp.Get().GetElements()[1].String(),
				Message:     resp.Get().GetElements()[2].String(),
			}
			break
		}
	}

	responseData, err := json.Marshal(respData)
	if err != nil {
		fmt.Println("Error encoding response:", err)
		return
	}

	_, err = conn.Write(responseData)
	if err != nil {
		fmt.Println("Error sending response:", err)
	}

	fmt.Printf("Sent response: %s\n", resp)
}

// worker
package main

import (
	"strconv"
	"time"
	ts "tuplespaceCD/pkg/tuplespace"
)

func worker(space *ts.Space) {
	for {
		reqChan := space.Get(ts.MakeTuple(ts.S("REQ"), ts.Any(), ts.Any(), ts.Any(), ts.Any(), ts.Any()))
		req := <-reqChan // Read from the channel
		if req != nil {
			bankAccount := req.Get().GetElements()[1].String()
			password := req.Get().GetElements()[2].String()
			requisition := req.Get().GetElements()[3].String()
			requisitionData := req.Get().GetElements()[4].String()

			switch requisition {
			case "create":
				space.Write(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.I(0)))
				space.Write(ts.MakeTuple(ts.S("RESP"), ts.S(bankAccount), ts.S("Account created")))

			case "delete":
				space.Get(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.Any()))
				space.Write(ts.MakeTuple(ts.S("RESP"), ts.S(bankAccount), ts.S("Account deleted")))

			case "deposit":
				tupleChan := space.Get(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.Any()))
				tuple := <-tupleChan
				if tuple != nil {
					moneyStr := tuple.Get().GetElements()[2].String()
					money, _ := strconv.Atoi(moneyStr)
					depositAmount, _ := strconv.Atoi(requisitionData)
					space.Write(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.I(money+depositAmount)))
					space.Write(ts.MakeTuple(ts.S("RESP"), ts.S(bankAccount), ts.S("Deposit successful")))
				} else {
					space.Write(ts.MakeTuple(ts.S("RESP"), ts.S(bankAccount), ts.S("Account not found")))
				}

			case "withdraw":
				tupleChan := space.Get(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.Any()))
				tuple := <-tupleChan
				if tuple != nil {
					moneyStr := tuple.Get().GetElements()[2].String()
					money, _ := strconv.Atoi(moneyStr)
					withdrawAmount, _ := strconv.Atoi(requisitionData)
					if money >= withdrawAmount {
						space.Write(ts.MakeTuple(ts.S(bankAccount), ts.S(password), ts.I(money-withdrawAmount)))
						space.Write(ts.MakeTuple(ts.S("RESP"), ts.S(bankAccount), ts.S("Withdrawal successful")))
					} else {
						space.Write(ts.MakeTuple(ts.S("RESP"), ts.S(bankAccount), ts.S("Insufficient funds")))
					}
				} else {
					space.Write(ts.MakeTuple(ts.S("RESP"), ts.S(bankAccount), ts.S("Account not found")))
				}
			}
		}
		time.Sleep(1 * time.Second) // Sleep for a second
	}
}

func main() {
	space := ts.NewSpace()
	go worker(space)
	select {}
}

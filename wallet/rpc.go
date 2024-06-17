package wallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/shopspring/decimal"
	"github.com/valyala/fastjson"
)

type RpcContent struct {
	Jsonrpc string      `json:"jsonrpc"`
	Id      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

func (s *Server) authenticatedRequest(rpcContent *RpcContent) ([]byte, error) {
	// Encode RPC request
	requestContent, _ := json.Marshal(rpcContent)

	// Build RPC request
	rpcRequest, _ := http.NewRequest("POST", fmt.Sprintf("https://%s:%d/", s.RpcAddress, s.RpcPort), bytes.NewReader(requestContent))
	rpcRequest.SetBasicAuth(s.RpcUser, s.RpcPass)
	rpcRequest.Header.Add("Content-Type", "application/json")
	rpcRequest.Header.Set("User-Agent", "PKT-Checkout")

	// Send RPC request
	response, err := s.RpcClient.Do(rpcRequest)
	if err != nil {
		return nil, err
	}

	// Return RPC request response
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (s *Server) getWalletAddresses() ([]string, error) {
	// Request
	content := RpcContent{
		Jsonrpc: "1.0",
		Id:      "mantpool",
		Method:  "getaddressbalances",
		Params:  &[]int{0, 1},
	}

	// Send the request
	request, err := s.authenticatedRequest(&content)
	if err != nil {
		return nil, err
	}

	// Parse the response
	data, err := fastjson.Parse(string(request))
	if err != nil {
		return nil, err
	}

	// Build return data
	var addresses []string
	results := data.GetArray("result")
	for _, entry := range results {
		addresses = append(addresses, string(entry.GetStringBytes("address")))
	}

	return addresses, nil
}

func (s *Server) getNewAddress() (string, error) {
	// Request
	content := RpcContent{
		Jsonrpc: "1.0",
		Id:      "mantpool",
		Method:  "getnewaddress",
		Params:  &[]int{0},
	}

	// Send the request
	request, err := s.authenticatedRequest(&content)
	if err != nil {
		return "", err
	}

	// Parse the response
	data, err := fastjson.Parse(string(request))
	if err != nil {
		return "", err
	}

	// Build return data
	return string(data.GetStringBytes("result")), nil
}

func (s *Server) getTransactions() ([]BlockchainTransaction, error) {
	// Request
	content := RpcContent{
		Jsonrpc: "1.0",
		Id:      "mantpool",
		Method:  "listtransactions",
		Params:  &[]int{500, 0},
	}

	// Send the request
	request, err := s.authenticatedRequest(&content)
	if err != nil {
		return nil, err
	}

	// Parse the response
	data, err := fastjson.Parse(string(request))
	if err != nil {
		return nil, err
	}

	// Build return data
	var transactions []BlockchainTransaction
	results := data.GetArray("result")
	for _, tx := range results {
		var transaction BlockchainTransaction

		floatAmount := decimal.NewFromFloat(tx.GetFloat64("amount"))
		microAmount := floatAmount.Mul(decimal.NewFromInt(1000000)).BigInt().Uint64()

		transaction.Id = string(tx.GetStringBytes("txid"))
		transaction.WalletAddress = string(tx.GetStringBytes("address"))
		transaction.PaymentAmount = microAmount
		transaction.DiscoveryTime = tx.GetUint64("time")
		transaction.Confirmations = uint32(tx.GetUint("confirmations"))

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

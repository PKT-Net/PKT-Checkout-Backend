package callback

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"pkt-checkout/database"
	"time"
)

type CallbackContent struct {
	Id        string           `json:"id"`
	Signature string           `json:"signature"`
	Invoice   database.Invoice `json:"invoice"`
}

func (s *Server) sendCallbackRequest(callback database.Callback) {
	// Fetch the corresponding invoice from database
	invoice, err := database.FetchInvoiceById(callback.InvoiceId)
	if err != nil {
		s.failedCallbackRequest(callback)
		return
	}

	// Fetch the corresponding account from database
	account, err := database.FetchAccountById(invoice.AccountId)
	if err != nil {
		s.failedCallbackRequest(callback)
		return
	}

	// Sign the ID for HMAC authentication by recipient
	h := hmac.New(sha256.New, []byte(account.SecretKey))
	h.Write([]byte(callback.Id))
	signature := hex.EncodeToString(h.Sum(nil))

	// Assemeble the content to transmit
	var callbackContent CallbackContent
	callbackContent.Id = callback.Id
	callbackContent.Signature = signature
	callbackContent.Invoice = invoice

	// Encode to JSON
	encodedContent, err := json.Marshal(callbackContent)
	if err != nil {
		s.failedCallbackRequest(callback)
		return
	}

	// Attempt sending request
	response, err := http.Post(invoice.CallbackUrl, "application/json", bytes.NewReader(encodedContent))
	if err != nil || response.StatusCode != 200 {
		s.failedCallbackRequest(callback)
		return
	}

	// OK
	callback.Status = database.CallbackStatusDelivered
	callback.Update()
}

func (s *Server) failedCallbackRequest(callback database.Callback) {
	// Retry at a later time, or give up
	callback.ReqErrors++
	if callback.ReqErrors >= s.Attempts {
		callback.Status = database.CallbackStatusFailed
		callback.Update()
		return
	}
	callback.NextReqTime = time.Now().Add(time.Duration(callback.ReqErrors) * (time.Duration(s.Backoff) * time.Minute))
	callback.Update()
}

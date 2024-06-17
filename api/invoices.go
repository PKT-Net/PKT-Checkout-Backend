package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"pkt-checkout/database"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (s *Server) getInvoiceById(c *fiber.Ctx) error {
	// Fetch account for apiKey
	apiKey := string(c.Request().Header.Peek("X-API-KEY"))
	account, err := database.FetchAccountByApiKey(apiKey)
	if err != nil {
		c.Response().SetStatusCode(403)
		return c.JSON(craftApiError("authentication_error", "Provided apiKey matches no account"))
	}

	// Fetch invoice for invoiceId
	invoiceId := c.Params("id")
	invoice, err := database.FetchInvoiceById(invoiceId)
	if err != nil || invoice.AccountId != account.Id {
		c.Response().SetStatusCode(403)
		return c.JSON(craftApiError("authentication_error", "Provided invoiceId matches no invoice"))
	}

	return c.JSON(invoice)
}

func (s *Server) getInvoicePublicById(c *fiber.Ctx) error {
	// Fetch account for viewKey
	viewKey := string(c.Request().Header.Peek("X-VIEW-KEY"))
	account, err := database.FetchAccountByViewKey(viewKey)
	if err != nil {
		c.Response().SetStatusCode(403)
		return c.JSON(craftApiError("authentication_error", "Provided viewKey matches no account"))
	}

	// Fetch invoice for invoiceId
	invoiceId := c.Params("id")
	invoice, err := database.FetchInvoiceById(invoiceId)
	if err != nil || invoice.AccountId != account.Id {
		c.Response().SetStatusCode(403)
		return c.JSON(craftApiError("authentication_error", "Provided invoiceId matches no invoice"))
	}

	// Add CORS header
	if len(s.CorsOrigin) > 0 {
		c.Response().Header.Add("Access-Control-Allow-Origin", s.CorsOrigin)
		c.Response().Header.Add("Access-Control-Allow-Headers", "X-VIEW-KEY")
	}

	return c.JSON(struct {
		Id                 string                 `json:"id"`
		PaymentAmount      uint64                 `json:"paymentAmount"`
		PaymentAddress     string                 `json:"paymentAddress"`
		PaymentDescription string                 `json:"paymentDescription"`
		ExpirationTime     time.Time              `json:"expirationTime"`
		Status             database.InvoiceStatus `json:"status"`
	}{
		Id:                 invoice.Id,
		PaymentAmount:      invoice.PaymentAmount,
		PaymentAddress:     invoice.PaymentAddress,
		PaymentDescription: invoice.PaymentDescription,
		ExpirationTime:     invoice.ExpirationTime,
		Status:             invoice.Status,
	})
}

func (s *Server) createInvoice(c *fiber.Ctx) error {
	// Fetch account for apiKey
	apiKey := string(c.Request().Header.Peek("X-API-KEY"))
	account, err := database.FetchAccountByApiKey(apiKey)
	if err != nil {
		c.Response().SetStatusCode(403)
		return c.JSON(craftApiError("authentication_error", "Provided apiKey matches no account"))
	}

	// Validate the signature
	hexSignature, err := hex.DecodeString(string(c.Request().Header.Peek("X-SIGNATURE")))
	if err != nil {
		c.Response().SetStatusCode(403)
		return c.JSON(craftApiError("authentication_error", "Provided signature matches no account"))
	}

	// Generate HMAC of content
	h := hmac.New(sha256.New, []byte(account.SecretKey))
	h.Write(c.Request().Body())
	if !bytes.Equal(h.Sum(nil), hexSignature) {
		c.Response().SetStatusCode(403)
		return c.JSON(craftApiError("authentication_error", "Provided signature matches no account"))
	}

	// Expected arguments
	var arguments struct {
		ClientId           string `json:"clientId"`
		PaymentAmount      uint64 `json:"paymentAmount"`
		PaymentDescription string `json:"paymentDescription"`
		PaymentExpiration  uint16 `json:"paymentExpiration"`
		CallbackUrl        string `json:"callbackUrl"`
	}
	if err = json.Unmarshal(c.Request().Body(), &arguments); err != nil {
		c.Response().SetStatusCode(400)
		return c.JSON(craftApiError("processing_error", "Provided request body unexpected"))
	}

	// Validate client ID
	if len(arguments.ClientId) > 0 {
		if len(arguments.ClientId) > 36 {
			c.Response().SetStatusCode(400)
			return c.JSON(craftApiError("processing_error", "Invoice client ID must be less than 37 chars"))
		}
	}

	// Validate payment amount
	if arguments.PaymentAmount < 1 {
		c.Response().SetStatusCode(400)
		return c.JSON(craftApiError("processing_error", "Invoice payment amount must be greater than 0 µPKT"))
	}

	// Validate payment description
	if len(arguments.PaymentDescription) > 0 {
		if !regexp.MustCompile("^[A-Za-z0-9 :-]+$").MatchString(arguments.PaymentDescription) {
			c.Response().SetStatusCode(400)
			return c.JSON(craftApiError("processing_error", "Invoice payment description must match regex ^[A-Za-z0-9 :-]+$"))
		}
	}

	// Validate payment expiration
	if arguments.PaymentExpiration > 0 {
		if arguments.PaymentExpiration < 5 || arguments.PaymentExpiration > 60 {
			c.Response().SetStatusCode(400)
			return c.JSON(craftApiError("processing_error", "Invoice payment expiration must be within 5 to 60 minutes"))
		}
	}

	// Validate callback URL
	if len(arguments.CallbackUrl) > 0 {
		if uri, err := url.ParseRequestURI(arguments.CallbackUrl); err != nil || uri.Scheme != "https" {
			c.Response().SetStatusCode(400)
			return c.JSON(craftApiError("processing_error", "Invoice callback URL must be valid URL"))
		}
	}

	// Fetch payment address
	paymentAddress, err := database.FetchLRUWalletAddress()
	if err != nil {
		c.Response().SetStatusCode(500)
		return c.JSON(craftApiError("processing_error", "Internal processing error"))
	}

	// Build invoice
	var invoice database.Invoice
	invoice.Id = uuid.New().String()
	invoice.ClientId = arguments.ClientId
	invoice.AccountId = account.Id
	invoice.PaymentAmount = arguments.PaymentAmount
	invoice.PaymentAddress = paymentAddress
	invoice.PaymentDescription = arguments.PaymentDescription
	invoice.CallbackUrl = arguments.CallbackUrl
	invoice.CreationTime = time.Now()
	if arguments.PaymentExpiration > 0 {
		invoice.ExpirationTime = time.Now().Add(time.Duration(arguments.PaymentExpiration) * time.Minute)
	} else {
		invoice.ExpirationTime = time.Now().Add(time.Duration(s.InvoiceTimeout) * time.Minute)
	}
	invoice.Status = database.InvoiceStatusCreated

	if err = invoice.Save(); err != nil {
		c.Response().SetStatusCode(500)
		return c.JSON(craftApiError("processing_error", "Internal processing error"))
	}

	return c.JSON(invoice)
}

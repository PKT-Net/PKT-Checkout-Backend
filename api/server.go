package api

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Server struct {
	HttpAddress    string
	HttpPort       uint16
	CorsOrigin     string
	InvoiceTimeout int
}

func NewServer() *Server {
	server := Server{
		HttpAddress:    viper.GetString("api-http-address"),
		HttpPort:       viper.GetUint16("api-http-port"),
		CorsOrigin:     "",
		InvoiceTimeout: 15,
	}

	if viper.IsSet("api-invoice-timeout") {
		server.InvoiceTimeout = viper.GetInt("api-invoice-timeout")
	}

	if viper.IsSet("api-cors-origin") {
		server.CorsOrigin = viper.GetString("api-cors-origin")
	}

	return &server
}

func (s *Server) Start() {
	app := fiber.New(fiber.Config{
		AppName:               "pkt-checkout",
		EnableIPValidation:    true,
		DisableStartupMessage: true,
	})

	// GET requests
	app.Get("v1/invoices/:id", s.getInvoiceById)
	app.Get("/v1/invoices/view/:id", s.getInvoicePublicById)
	app.Options("/v1/invoices/view/:id", s.preflightPublicView)

	// POST requests
	app.Post("/v1/invoices", s.createInvoice)

	log.Info().Msg("Starting HTTP API server")
	if err := app.Listen(fmt.Sprintf("%s:%d", s.HttpAddress, s.HttpPort)); err != nil {
		log.Fatal().Err(err).Msg("Starting HTTP API server failed")
	}
}

package main

import (
	"fmt"
	"pkt-checkout/api"
	"pkt-checkout/callback"
	"pkt-checkout/database"
	"pkt-checkout/wallet"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func requiredConfigParams() []string {
	return []string{"api-http-address",
		"api-http-port",
		"mysql-address",
		"mysql-port",
		"mysql-database",
		"mysql-user",
		"mysql-pass",
		"wallet-rpc-address",
		"wallet-rpc-port",
		"wallet-rpc-user",
		"wallet-rpc-pass",
		"wallet-addresses",
		"wallet-confirmations",
		"callback-attempts",
		"callback-backoff"}
}

func main() {
	// Read configuration
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if viper.ReadInConfig() != nil {
		log.Fatal().Msg("Reading configuration file failed")
	}

	// Lazy-validate configuration
	for _, key := range requiredConfigParams() {
		if !viper.IsSet(key) {
			log.Fatal().Str("error", fmt.Sprintf("missing configuration key: %s", key)).Msg("Interpreting configuration file failed")
		}
	}

	// Start the database server
	databaseServer := database.NewServer()
	databaseServer.Start()

	// Start the wallet server
	walletServer := wallet.NewServer()
	go walletServer.Start()

	// Start the callback server
	callbackServer := callback.NewServer()
	go callbackServer.Start()

	// Start the API server
	apiServer := api.NewServer()
	go apiServer.Start()

	// Run forever
	<-make(chan int)
}

package wallet

import (
	"crypto/tls"
	"net/http"
	"pkt-checkout/database"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Server struct {
	RpcClient       *http.Client
	RpcAddress      string
	RpcPort         uint16
	RpcUser         string
	RpcPass         string
	TxAddresses     int
	TxConfirmations uint32
}

func NewServer() *Server {
	return &Server{
		RpcClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		RpcAddress:      viper.GetString("wallet-rpc-address"),
		RpcPort:         viper.GetUint16("wallet-rpc-port"),
		RpcUser:         viper.GetString("wallet-rpc-user"),
		RpcPass:         viper.GetString("wallet-rpc-pass"),
		TxAddresses:     viper.GetInt("wallet-addresses"),
		TxConfirmations: viper.GetUint32("wallet-confirmations"),
	}
}

func (s *Server) Start() {
	db := database.GetConnection()

	// Fetch addresses from wallet
	walletAddresses, err := s.getWalletAddresses()
	if err != nil {
		log.Fatal().Err(err).Msg("Fetching addresses from wallet backend failed")
	}

	// Fetch addresses from database
	dbResults, err := db.Query("SELECT address FROM walletAddresses")
	if err != nil {
		log.Fatal().Err(err).Msg("Fetching addresses from database backend failed")
	}

	// Loading the data
	var dbAddresses []string
	for dbResults.Next() {
		var address string
		dbResults.Scan(&address)
		dbAddresses = append(dbAddresses, address)
	}

	// Consistency check
	for _, dbAddress := range dbAddresses {
		addressFound := false
		for _, walletAddress := range walletAddresses {
			if dbAddress == walletAddress {
				addressFound = true
				break
			}
		}
		if !addressFound {
			log.Fatal().Err(err).Msg("Inconsistency between wallet backend and database backend")
		}
	}

	// Generate missing addresses
	if len(dbAddresses) < s.TxAddresses {
		for j := 0; j < (s.TxAddresses - len(dbAddresses)); j++ {
			// Wallet backend
			address, err := s.getNewAddress()
			if err != nil {
				log.Fatal().Err(err).Msg("Generating missing addresses failed")
			}

			// Database backend
			stmt, err := db.Prepare("INSERT INTO walletAddresses (address) VALUES (?)")
			if err != nil {
				log.Fatal().Err(err).Msg("Generating missing addresses failed")
			}
			_, err = stmt.Exec(address)
			if err != nil {
				log.Fatal().Err(err).Msg("Generating missing addresses failed")
			}
		}
	}

	for {
		s.Scan()
		time.Sleep(30 * time.Second)
	}
}

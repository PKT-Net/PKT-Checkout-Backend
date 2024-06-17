package callback

import (
	"pkt-checkout/database"
	"time"

	"github.com/spf13/viper"
)

type Server struct {
	Attempts int
	Backoff  int
}

func NewServer() *Server {
	return &Server{
		Attempts: viper.GetInt("callback-attempts"),
		Backoff:  viper.GetInt("callback-backoff"),
	}
}

func (s *Server) Start() {
	for {
		// Fetch pending callbacks from database
		callbacks, err := database.FetchPendingCallbacks()
		if err != nil {
			time.Sleep(30 * time.Second)
			continue
		}

		// Attempt delivering the callback
		for _, callback := range callbacks {
			s.sendCallbackRequest(callback)
		}

		time.Sleep(30 * time.Second)
	}
}

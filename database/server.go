package database

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var connection *sql.DB

type Server struct {
	MysqlAddress  string
	MysqlPort     uint16
	MysqlDatabase string
	MysqlUser     string
	MysqlPass     string
}

func NewServer() *Server {
	return &Server{
		MysqlAddress:  viper.GetString("mysql-address"),
		MysqlPort:     viper.GetUint16("mysql-port"),
		MysqlDatabase: viper.GetString("mysql-database"),
		MysqlUser:     viper.GetString("mysql-user"),
		MysqlPass:     viper.GetString("mysql-pass"),
	}
}

func (s *Server) Start() {
	// Prepare connection
	c, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", s.MysqlUser, s.MysqlPass, s.MysqlAddress, s.MysqlPort, s.MysqlDatabase))
	if err != nil {
		log.Fatal().Err(err).Msg("Creating database connection failed")
	}
	c.SetMaxOpenConns(100)

	// Validate connection
	if err = c.Ping(); err != nil {
		log.Fatal().Err(err).Msg("Establishing database connection failed")
	}

	connection = c
}

func GetConnection() *sql.DB {
	return connection
}

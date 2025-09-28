package config

import (
	"log/slog"

	"github.com/corray333/backend-labs/order/pkg/logger"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

func MustInit() {
	if err := godotenv.Load("./.env"); err != nil {
		panic("error while loading .env file: " + err.Error())
	}
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/order-svc")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic("error while reading config file: " + err.Error())
	}
	SetupLogger()
}

func SetupLogger() {
	handler := logger.NewHandler(nil)
	log := slog.New(handler)
	slog.SetDefault(log)
}

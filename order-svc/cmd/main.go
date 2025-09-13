package main

import (
	"github.com/corray333/backend-labs/order/internal/app"
	"github.com/corray333/backend-labs/order/internal/config"
)

func main() {
	config.MustInit()
	app.MustNewApp().Run()
}

package main

import (
	"github.com/corray333/backend-labs/consumer/internal/app"
	"github.com/corray333/backend-labs/consumer/internal/config"
)

func main() {
	config.MustInit()
	app.MustNewApp().Run()
}

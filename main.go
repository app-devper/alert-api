package main

import (
	"alert/app"
)

func main() {
	var server app.Routes
	server.StartGin()
}

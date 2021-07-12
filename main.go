package main

import (
	"flag"
)

func main() {
	var config string
	flag.StringVar(&config, "config", "", "configuration file")
	flag.Parse()
	server(config)
}

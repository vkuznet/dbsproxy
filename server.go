package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	_ "expvar"         // to be used for monitoring, see https://github.com/divan/expvarmon
	_ "net/http/pprof" // profiler, see https://golang.org/pkg/net/http/pprof/

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

// APIMapping keeps track of timestamps for given API name
type APIMapping struct {
	Api        string  `json:"api"`
	Timestamps []int64 `json:"timestamps"`
}

// Configuration stores server configuration parameters
type Configuration struct {
	Port         int          `json:"port"`         // server port number
	Base         string       `json:"base"`         // base URL
	Verbose      int          `json:"verbose"`      // verbose output
	LogFile      string       `json:"log_file"`     // log file
	UTC          bool         `json:"utc"`          // report logger time in UTC
	MonitRecord  bool         `json:"monit_record"` // print monit record on stdout
	Backends     []string     `json:"backends"`     // DBS backends
	Styles       string       `json:"styles"`       // CSS styles path
	Jscripts     string       `json:"js"`           // JS path
	Images       string       `json:"images"`       // images path
	ServerCrt    string       `json:"server_cert"`  // path to server crt file
	ServerKey    string       `json:"server_key"`   // path to server key file
	APIRedirects []APIMapping `json:"api_redirects"`
	BufferSize   int          `json:"scanner_buffer_size"` // buffer size of the scanner
}

// Config variable represents configuration object
var Config Configuration

// helper function to parse configuration
func parseConfig(configFile string) error {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Println("Unable to read", err)
		return err
	}
	err = json.Unmarshal(data, &Config)
	if err != nil {
		log.Println("Unable to parse", err)
		return err
	}
	if Config.BufferSize == 0 {
		Config.BufferSize = 1024 * 1024
	}
	return nil
}

// http server implementation
func server(config string) {
	err := parseConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	log.SetFlags(0)
	if Config.Verbose > 0 {
		log.SetFlags(log.Lshortfile)
	}
	log.SetOutput(new(logWriter))
	if Config.LogFile != "" {
		rl, err := rotatelogs.New(Config.LogFile + "-%Y%m%d")
		if err == nil {
			rotlogs := rotateLogWriter{RotateLogs: rl}
			log.SetOutput(rotlogs)
		}
	}

	if Config.Verbose > 0 {
		log.Printf("Config: %+v", Config)
	}

	// the proxy handler
	http.HandleFunc(fmt.Sprintf("%s/", Config.Base), ProxyHandler)

	// start HTTP or HTTPs server based on provided configuration
	addr := fmt.Sprintf(":%d", Config.Port)
	if Config.ServerCrt != "" && Config.ServerKey != "" {
		//start HTTPS server which require user certificates
		server := &http.Server{Addr: addr}
		log.Printf("Starting HTTPs server on %s", addr)
		log.Fatal(server.ListenAndServeTLS(Config.ServerCrt, Config.ServerKey))
	} else {
		// Start server without user certificates
		log.Printf("Starting HTTP server on %s", addr)
		log.Fatal(http.ListenAndServe(addr, nil))
	}
}

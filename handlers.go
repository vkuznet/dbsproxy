package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

// helper function to get random backend server
func getServer() string {
	idx := rand.Intn(len(Config.Backends))
	return Config.Backends[idx]
}

// helper function to send request to backend server
// func send(rurl string, w http.ResponseWriter, r *http.Request, wg *sync.WaitGroup) error {
func send(rurl string, ch chan []byte, r *http.Request, wg *sync.WaitGroup) error {
	start := time.Now()
	defer log.Println("send %s %v", r, time.Since(start))
	defer wg.Done()
	// send HTTP request to backend server
	client := http.Client{}
	if Config.Verbose > 0 {
		log.Println("send", rurl)
	}
	req, err := http.NewRequest("GET", rurl, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Accept", "application/ndjson")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	// we scan our ndjson records and send over individual JSON records to our channel
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer([]byte{}, Config.BufferSize)
	for scanner.Scan() {
		line := scanner.Text()
		ch <- []byte(line)
	}
	if err := scanner.Err(); err != nil {
		log.Println("scanner error", err)
		return err
	}

	return nil
}

// helper function to collect results from goroutines and write them to reponse writer
func collect(w http.ResponseWriter, ch chan []byte, terminate chan bool) {
	start := time.Now()
	defer log.Println("collect %v", time.Since(start))
	w.Write([]byte("[\n"))
	defer w.Write([]byte("]\n"))
	var sep bool
	for {
		select {
		case data := <-ch:
			if sep {
				w.Write([]byte(","))
			}
			w.Write(data)
			w.Write([]byte("\n"))
			sep = true
		case <-terminate:
			return
		default:
			time.Sleep(time.Duration(1) * time.Millisecond) // wait for response
		}
	}
}

// ProxyHandler provides basic functionality of status response
func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer log.Println("%s %s %v", r.Method, r.RequestURI, time.Since(start))

	// get random DBS Backend server
	srv := getServer()

	// for non GET requests we simply redirect
	if r.Method != "GET" {
		reverseProxy(srv, w, r)
		return
	}

	// extract api of the request
	arr := strings.Split(r.URL.Path, "/")
	api := arr[len(arr)-1]

	// find if our api maps contains proper set of timestamps
	var tstamps []int64
	for _, amap := range Config.APIRedirects {
		if amap.Api == api {
			tstamps = amap.Timestamps
			// add current timestamp as last entry in tstamps array
			tstamps = append(tstamps, time.Now().Unix())
			break
		}
	}
	// if timestamps found we'll adjust request URL, otherwise we'll simply redirect to BE server
	if len(tstamps) > 0 {
		// collect data from goroutines and organize JSON stream
		ch := make(chan []byte)      // this channel collects individual JSON records
		terminate := make(chan bool) // this channel terminates collect goroutine
		go collect(w, ch, terminate)

		// send goroutines to DBS backend servers
		var wg sync.WaitGroup
		var prev int64
		for _, ts := range tstamps {
			wg.Add(1)
			// construct new request url
			rurl := fmt.Sprintf("%s/%s?%s", srv, api, r.URL.RawQuery)
			// adjust raw url with new timestamp ranges
			if !strings.Contains(r.URL.RawQuery, "create_by") {
				rurl = fmt.Sprintf("%s/%s?%s&min_cdate=%d&max_cdate=%d", srv, api, r.URL.RawQuery, prev, ts)
			}
			go send(rurl, ch, r, &wg)
			prev = ts
		}
		wg.Wait()
		// once we received all goroutines send calls we'll terminate collect goroutine
		terminate <- true
	} else {
		reverseProxy(srv, w, r)
	}
}

func reverseProxy(rurl string, w http.ResponseWriter, r *http.Request) {
	nurl, err := url.Parse(rurl)
	if err != nil {
		log.Println("reverse proxy error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// ServeHttp is non blocking and uses a go routine under the hood
	proxy := httputil.NewSingleHostReverseProxy(nurl)
	proxy.ServeHTTP(w, r)
}

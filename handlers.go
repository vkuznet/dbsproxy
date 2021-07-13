package main

import (
	"fmt"
	"io"
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
func send(rurl string, w http.ResponseWriter, r *http.Request, wg *sync.WaitGroup) error {
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
	// write response directly to our writer
	defer resp.Body.Close()
	bytes, err := io.Copy(w, resp.Body)
	if Config.Verbose > 0 {
		log.Printf("%s receive %d bytes", rurl, bytes)
	}
	return err
}

// ProxyHandler provides basic functionality of status response
func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	// get random DBS Backend server
	srv := getServer()

	// for non GET requests we simply redirect
	if r.Method != "GET" {
		reverseProxy(srv, w, r)
		return
	}

	// extract api of the request
	arr := strings.Split(r.URL.Path, "/")
	log.Println("URL path", r.URL.Path, arr)
	api := arr[len(arr)-1]

	// find if our api maps contains proper set of timestamps
	var tstamps []int64
	for _, amap := range Config.APIRedirects {
		log.Println("amap", amap, api)
		if amap.Api == api {
			tstamps = amap.Timestamps
			// add current timestamp as last entry in tstamps array
			tstamps = append(tstamps, time.Now().Unix())
			break
		}
	}
	// if timestamps found we'll adjust request URL, otherwise we'll simply redirect to BE server
	if len(tstamps) > 0 {
		// for GET HTTP calls we'll use api redirect mapping
		w.Write([]byte("[\n"))
		defer w.Write([]byte("]\n"))

		// send goroutines to DBS backend servers
		var wg sync.WaitGroup
		var prev int64
		for _, ts := range tstamps {
			wg.Add(1)
			// construct new request url
			rurl := fmt.Sprintf("%s/%s?%s", srv, api, r.URL.RawQuery)
			// adjust raw url with new timestamp ranges
			if !strings.Contains(r.URL.RawQuery, "create_by") {
				rurl = fmt.Sprintf("%s/%s?%s&min_cdate=%d&max_cdate", srv, api, r.URL.RawQuery, prev, ts)
			}
			go send(rurl, w, r, &wg)
			prev = ts
		}
		wg.Wait()
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

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"

	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

const target = "https://www.google-analytics.com"  // reverse proxy target.
const tid = "UA-98437906-1"                        // identifies ga user property.
const cid = "0eaa001e-7b9e-44ef-997b-150eb33a01ff" // identifies this reverse proxy server.
const splitRegex = " /spl/ "

func main() {
	urlTarget, _ := url.Parse(target)
	proxy := NewSingleHostReverseProxy(urlTarget)

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		// Investigate further to find out who the requester is.
		ip, _, _ := net.SplitHostPort(req.RemoteAddr)
		ua := req.Header.Get("User-Agent")
		ea := fmt.Sprint(req.Method, splitRegex, req.RequestURI, splitRegex)
		for k, v := range req.Header {
			ea += fmt.Sprint(k, ":", v, splitRegex)
		}
		el := "Failed to read body. Reason: "
		body, err := ioutil.ReadAll(req.Body) // this consumes it all up
		if err != nil {
			el += err.Error()
		} else {
			el = string(body)
		}
		req.Body = ioutil.NopCloser(bytes.NewBuffer(body)) // assign another to be sucked up

		// Server side ga() call with the Measurement Protocol.
		// ref) https://developers.google.com/analytics/devguides/collection/protocol/v1/devguide
		param := url.Values{}
		param.Set("v", "1")
		param.Add("t", "event")
		param.Add("tid", tid)
		param.Add("cid", cid)
		param.Add("ec", "Request to Proxy v0.1")
		param.Add("ea", ea) // event action: req header
		param.Add("el", el) // event label: req body
		param.Add("uip", ip)
		param.Add("ua", ua)
		_, err = urlfetch.Client(appengine.NewContext(req)).PostForm(target+"/collect", param) // htmlutil.PostForm()
		if err != nil {
			log.Println("Measurement Protocol fails: ", err)
		}

		// Serving proxy.
		// This seems to be not working correctly but I can't really figure out why. It needs to be fixed later.
		proxy.ServeHTTP(w, req) // non blocking and uses a go routine under the hood
	})

	// log.Fatal(http.ListenAndServe(":8181", nil))
	appengine.Main() // Starts the server to receive requests
}

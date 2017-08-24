// client will serve as a proxy with basic authentication method. It will
// communicate with the underlying proxy, or send requests to remote proxy if
// the requests are blocked by the underlying proxy.
package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/ramuchu/fetch"

	"net/http"
	_ "net/http/pprof"
	"net/url"
)

var proxyURL string
var hostURL string
var localPort string
var useragent string

func getRemoteProxy() string {
	e := os.Getenv("REMOTE_PROXY")
	if e == "" {
		return "http://localhost:8000"
	}
	return e
}

func init() {
	flag.StringVar(&localPort, "port", "8282", "the port this server going to listen")
	flag.StringVar(&hostURL, "host", getRemoteProxy(), "Address of Remote server, $REMOTE_PROXY if set")
	flag.StringVar(&proxyURL, "proxy", os.Getenv("HTTP_PROXY"), "Address of HTTP proxy server, $HTTP_PROXY if set")
	flag.StringVar(&useragent, "agent", os.Getenv("AGENT"), "UserAgent of HTTP requests, $AGENT if set")
}

func main() {
	flag.Parse()
	host := "localhost"
	port := "8000"
	proto := "https"

	// parse the hostURL
	sch := strings.SplitN(hostURL, "://", 2)
	if len(sch) > 1 {
		proto = sch[0]
	}
	addr := sch[len(sch)-1]
	l := strings.Split(addr, ":")
	host = l[0]
	if len(l) > 1 {
		port = l[1]
	}

	var origin, pURL string
	switch proto {
	case "http":
		origin = "http://" + host + "/"
		pURL = "ws://" + host + ":" + port + "/p"
	case "https":
		origin = "https://" + host + "/"
		pURL = "wss://" + host + ":" + port + "/p"
	default:
		fmt.Println("Unknown protocol")
		return
	}

	fmt.Printf("Address of the websocket to connect to: [%s]\n", pURL)
	if proxyURL == "" {
		fmt.Println("Without http proxy")
	} else {
		fmt.Printf("With http proxy %s\n", proxyURL)
	}

	// handler to ask local proxy
	proxy := createProxy(proxyURL, useragent)
	proxyHandler := LogHandler("NTLMProxy  <--", proxy)

	// handler to ask remote proxy
	remoteProxy := LogHandler("Remote     <--", createRemoteProxy(proxy, pURL, "", origin))

	// cache handler
	hmap := map[string]http.Handler{
		"proxy":  proxyHandler,
		"remote": remoteProxy,
		"block": LogHandler("block       <--", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s := http.StatusForbidden
			http.Error(w, http.StatusText(s), s)
		})),
	}
	// read / write to the file
	f, err := os.OpenFile("data.txt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	cache := NewCacheHandler(nil, hmap, f)
	cache.AutoSaveTo(f)

	// default handler, which is also a local proxy, but will validate the result.
	// Whenever it finds the request is blocked, store the host to cache and fallback to remote
	defProxy := createProxy(proxyURL, useragent)
	defProxy.Fallback = remoteProxy
	defProxy.ValidHTTP = func(req *http.Request, resp *http.Response) error {
		err := validHTTP(req, resp)
		if err == nil {
			//go cache.Set(req.Host, "", proxyHandler)
		} else {
			go cache.Set(req.Host, "remote", remoteProxy)
		}
		return err
	}
	defProxy.ValidConnect = func(req *http.Request, c net.Conn) error {
		err := handshakeConnect(req.URL, c)
		if err == nil {
			// handshaking will make the client establish the connection once more
			// just remember it when it is running
			go cache.Set(req.Host, "", proxyHandler)
		} else {
			go cache.Set(req.Host, "remote", remoteProxy)
		}
		return err
	}
	cache.Default = LogHandler("           <--", defProxy)

	// start handling requests
	fmt.Println("Start listening", ":"+localPort)
	err = http.ListenAndServe(":"+localPort, cache)
	if err != nil {
		panic(err)
	}
}

func createProxy(proxyURL string, agent string) *NTLMProxy {
	proxy, err := NewNTLMProxy(proxyURL)
	if err != nil {
		panic(err)
	}
	proxy.Agent = agent
	return proxy
}

var errNotFound = errors.New("Not found")
var errFiltered = errors.New("Filtered")

// check if the request is blocked.
// just check if the response is redirecting to a known host
func validHTTP(req *http.Request, resp *http.Response) error {
	switch {
	case resp.StatusCode >= 400:
		return errNotFound
	case resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotModified:
		l := resp.Header.Get("Location")
		u, err := url.Parse(l)
		if err != nil {
			return err
		}
		if u.Host == "alert.scansafe.net" {
			return errFiltered
		}
	}
	return nil
}

func hasPort(s string) bool { return strings.LastIndex(s, ":") > strings.LastIndex(s, "]") }

// handshakeConnect sends a tls handshake to the connection.
// If the reply is not valid (e.g access denied), then we assume it is blocked
func handshakeConnect(u *url.URL, c net.Conn) error {
	h := u.Host
	if hasPort(h) {
		h = h[:strings.LastIndex(h, ":")]
	}
	cfg := &tls.Config{ServerName: h}
	tlsConn := tls.Client(c, cfg)
	return tlsConn.Handshake()
}

// createRemoteProxy use the NTLMProxy to establish a websocket connection, tunnel to remote server
func createRemoteProxy(proxy *NTLMProxy, pURL, protocol, origin string) http.Handler {
	genConn := func() (net.Conn, error) {
		//conn, err := ProxyDial(pURL, "", origin)
		conn, err := proxy.Websocket(pURL, "", origin)
		if err != nil {
			return nil, err
		}

		return fetch.NewClientConn(conn.UnderlyingConn(), 0x56), nil
	}
	genConn = logConnect(genConn)
	return Tunnel(genConn)
}

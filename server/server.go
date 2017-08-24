package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/kalokng/fetch"

	_ "net/http/pprof"

	"golang.org/x/net/websocket"
)

var echoWs = websocket.Handler(func(ws *websocket.Conn) {
	os.Stdout.Write([]byte("Start ECHO"))
	defer os.Stdout.Write([]byte("End ECHO"))
	r := io.TeeReader(ws, os.Stdout)
	io.Copy(ws, r)
})

func EchoServer(w http.ResponseWriter, r *http.Request) {
	echoWs.ServeHTTP(w, r)
}

func WebServer(w http.ResponseWriter, r *http.Request) {
	val := r.URL.Query()
	q := val.Get("q")
	if q == "" {
		q = "http://httpbin.org/ip"
	}
	resp, err := http.Get(q)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	resp.Write(w)
}

func serveGET(ws *websocket.Conn, req *http.Request) {
	fmt.Println("req.RequestURI", req.RequestURI)
	req.RequestURI = ""
	fmt.Println("req.URL.Scheme", req.URL.Scheme)
	req.URL.Scheme = "http"
	fmt.Println("req.URL.Host", req.URL.Host)
	req.URL.Host = req.Host
	fmt.Println("URL", req.URL.RequestURI())

	resp, err := http.DefaultTransport.(*http.Transport).RoundTrip(req)
	if err != nil {
		fmt.Println(err)
		io.WriteString(ws, "HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\n400 Bad Request: "+err.Error())
		return
	}
	resp.Write(ws)
}

func serveCONNECT(ws *websocket.Conn, req *http.Request) {
	host := req.URL.Host
	fmt.Println("CONNECTING", host, "...")
	c, err := http.DefaultTransport.(*http.Transport).Dial("tcp", host)
	if err != nil {
		fmt.Println("ERR:", err)
		io.WriteString(ws, "HTTP/1.1 500 Internal Server Error\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\n500 Internal Server Error: "+err.Error())
		return
	}
	fmt.Println("start tunnel...")
	go func() {
		ew := fetch.NewEncoder(ws)
		ew.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		io.Copy(ew, c)
		//ew.Close()
		ws.Close()
	}()
	er := fetch.NewDecoder(ws)
	io.Copy(c, er)
	c.Close()
}

var wsProxy = websocket.Handler(func(ws *websocket.Conn) {
	req, err := http.ReadRequest(bufio.NewReader(ws))
	if err != nil {
		io.WriteString(ws, "HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\n400 Bad Request")
		return
	}
	//b, _ := httputil.DumpRequestOut(req, true)
	//os.Stdout.Write(b)

	fmt.Println("req.Method", req.Method)
	switch req.Method {
	case "CONNECT":
		serveCONNECT(ws, req)
	default:
		serveGET(ws, req)
	}
	//io.WriteString(ws, "HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\n400 Bad Request")
	//return
})

func main() {
	http.HandleFunc("/echo", EchoServer)
	http.HandleFunc("/web", WebServer)
	http.Handle("/proxy", wsProxy)
	//proxy := NewProxyListener(nil)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Hello world")
		fmt.Fprintf(w, "Hello world!")
	})

	bind := getIP() + ":" + getPort()

	fmt.Println("Listening to", bind)
	err := http.ListenAndServe(bind, nil)
	if err != nil {
		panic(err)
	}
}

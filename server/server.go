package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/ramuchu/fetch"

	_ "net/http/pprof"
)

func EchoServer2(w http.ResponseWriter, r *http.Request) {
	fmt.Println("echo2")
	c, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
     	panic("cannot hijack http")
	}
	defer c.Close()

	tee := io.TeeReader(c, os.Stdout)
	c.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))
	fmt.Println("start echo2...")
	io.Copy(c, tee)
}

func EchoServer3(w http.ResponseWriter, r *http.Request) {
	fmt.Println("echo3")
	c, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		panic("cannot hijack http")
	}
	defer c.Close()

	tee := io.TeeReader(c, os.Stdout)
	c.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))
	fmt.Println("start echo2...")
	ibuf := make([]byte, 1024)
	obuf := make([]byte, 2*1024)
	var n int
	var ierr, oerr error
	for ierr == nil && oerr == nil {
		n, ierr = tee.Read(ibuf)
		n = hex.Encode(obuf, ibuf[:n])
		_, oerr = c.Write(obuf[:n])
	}
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

func WebServer2(w http.ResponseWriter, r *http.Request) {
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
	pushResponse(w, resp)
}

func serveGET(ws net.Conn, req *http.Request) {
	fmt.Println("req.RequestURI", req.RequestURI)
	req.RequestURI = ""
	fmt.Println("req.URL.Scheme", req.URL.Scheme)
	req.URL.Scheme = "http"
	fmt.Println("req.URL.Host", req.URL.Host)
	req.URL.Host = req.Host
	fmt.Println("URL", req.URL.RequestURI())

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		fmt.Println(err)
		io.WriteString(ws, "HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\n400 Bad Request: "+err.Error())
		return
	}
	resp.Write(ws)
}

func serveCONNECT(ws net.Conn, req *http.Request) {
	host := req.URL.Host
	fmt.Println("CONNECTING", host, "...")
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		fmt.Println("ERR: cannot resolve DefaultTransport")
		io.WriteString(ws, "HTTP/1.1 500 Internal Server Error\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\n500 Internal Server Error: cannot resolve DefaultTransport")
		return
	}
	c, err := transport.DialContext(req.Context(), "tcp", host)
	if err != nil {
		fmt.Println("ERR:", err)
		io.WriteString(ws, "HTTP/1.1 500 Internal Server Error\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\n500 Internal Server Error: "+err.Error())
		return
	}
	ws.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))
	fmt.Println("start tunnel...")
	go func() {
		io.Copy(ws, c)
		ws.Close()
	}()
	io.Copy(c, ws)
	c.Close()
}

var upgrader = websocket.Upgrader{}

func wsProxy(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("wsProxy error", err)
		return
	}

	conn := fetch.NewServerConn(ws.UnderlyingConn(), 0x56)
	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		io.WriteString(conn, "HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\n400 Bad Request")
		return
	}
	//b, _ := httputil.DumpRequestOut(req, true)
	//os.Stdout.Write(b)

	fmt.Println("req.Method", req.Method)
	switch req.Method {
	case "CONNECT":
		serveCONNECT(conn, req)
	default:
		serveGET(conn, req)
	}
}

func main() {
	http.HandleFunc("/echo3", EchoServer3)
	http.HandleFunc("/web", WebServer)
	http.HandleFunc("/web2", WebServer2)
	http.HandleFunc("/p", wsProxy)
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

func pushResponse(w http.ResponseWriter, resp *http.Response) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	resp.Write(conn)
}

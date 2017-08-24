package main

import (
	"io"
	"log"
	"net/http"
)

// LogHandler is an adapter which prints a log with prefix, the request method and host.
func LogHandler(prefix string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		go log.Printf("%s %s: %s", prefix, r.Method, r.URL.Host)
		h.ServeHTTP(w, r)
	})
}

// Tunnel returns a http.Handler, that whenever a request comes from client, it
// will create a conn from pool, and copy what they send and receive to each
// other.
//
// Tunnel serve as a middle man between client and remote side. From client
// point of view, it looks like it talks to the remote side.
func Tunnel(pool funcConn) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := pool()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		//{
		//	if b, err := httputil.DumpRequest(r, false); err == nil {
		//		fmt.Printf("%s\n", b)
		//	}
		//}

		//fmt.Fprintln(conn, "Received", r.URL.Path)
		hj, ok := w.(http.Hijacker)
		if !ok {
			panic("CANNOT hijack")
		}
		c, _, err := hj.Hijack()
		if err != nil {
			panic(err)
		}

		//mw := io.MultiWriter(conn, os.Stdout)
		// send out the request
		if r.Method == "CONNECT" {
			go func() {
				r.Write(conn)
				//io.Copy(conn, io.TeeReader(c, os.Stdout))
				io.Copy(conn, c)
				conn.Close()
			}()
			//io.Copy(io.MultiWriter(c, os.Stdout), conn)
			io.Copy(c, conn)
			c.Close()
			return
		}

		go r.Write(conn)

		io.Copy(c, conn)
		//io.Copy(conn, c)
		c.Close()
		conn.Close()
	})
}

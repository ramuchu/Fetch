package main

import (
	"log"
	"net"
)

type funcConn func() (net.Conn, error)

func logConnect(fn funcConn) funcConn {
	return func() (net.Conn, error) {
		conn, err := fn()
		if err != nil {
			log.Printf("connect err: %s\n", err.Error())
		}
		return conn, err
	}
}

/* Comment the unused functions
func HttpConnect(proxy, url_ string) (net.Conn, error) {
	p, err := net.Dial("tcp", proxy)
	if err != nil {
		return nil, err
	}

	turl, err := url.Parse(url_)
	if err != nil {
		return nil, err
	}

	req := http.Request{
		Method: "CONNECT",
		URL:    &url.URL{},
		Host:   turl.Host,
	}

	cc := httputil.NewProxyClientConn(p, nil)
	cc.Do(&req)
	if err != nil && err != httputil.ErrPersistEOF {
		return nil, err
	}

	rwc, _ := cc.Hijack()

	return rwc, nil
}

func ProxyDial(url_, protocol, origin string) (ws *websocket.Conn, err error) {
	if proxyURL == "" {
		return websocket.Dial(url_, protocol, origin)
	}

	purl, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	config, err := websocket.NewConfig(url_, origin)
	if err != nil {
		return nil, err
	}

	if protocol != "" {
		config.Protocol = []string{protocol}
	}

	client, err := HttpConnect(purl.Host, url_)
	if err != nil {
		return nil, err
	}

	return websocket.NewClient(config, client)
}

func ProxyHTTP(url_ string) (c net.Conn, err error) {
	if proxyURL == "" {
		fmt.Println("url_", url_)
		turl, err := url.Parse(url_)
		if err != nil {
			return nil, err
		}
		fmt.Println("turl", turl)

		fmt.Println("turl.Host", turl.Host)
		p, err := net.Dial("tcp", turl.Host)

		req := http.Request{
			Method: "CONNECT",
			URL:    turl,
			Host:   turl.Host,
		}

		cc := httputil.NewClientConn(p, nil)
		cc.Do(&req)
		if err != nil && err != httputil.ErrPersistEOF {
			return nil, err
		}

		rwc, _ := cc.Hijack()

		return rwc, nil
	}

	purl, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	return HttpConnect(purl.Host, url_)
}
*/

package main

import (
	"bytes"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	out := make(chan string, 1)
	h := make(map[string]http.Handler, 5)
	for _, v := range strings.Split("12345", "") {
		v := v
		h[v] = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			out <- v
		})
	}
	d := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out <- "def"
	})

	c := NewCacheHandler(d, h, nil)
	f, err := os.OpenFile("test_data.txt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	c.AutoSaveTo(f)
	for k, v := range h {
		c.Set(k+k, k, v)
	}
	req := &http.Request{URL: &url.URL{}}
	for _, v := range strings.Split("00;11;22;33;44;55;66;0;1;2;3;4;5;6", ";") {
		req.Host = v
		req.URL.Host = v
		c.ServeHTTP(nil, req)
		select {
		case s := <-out:
			if v[:1] != s {
				switch s {
				case "def":
				default:
					t.Fatalf("%s : %s", v, s)
				}
			}
		default:
			t.Fatal("No out when: " + v)
		}
	}
	var obuf bytes.Buffer
	if _, err := c.Save(&obuf); err != nil {
		t.Fatal(err)
	}

	nc := NewCacheHandler(d, h, &obuf)
	for _, v := range strings.Split("00;11;22;33;44;55;66;0;1;2;3;4;5;6", ";") {
		req.Host = v
		nc.ServeHTTP(nil, req)
		select {
		case s := <-out:
			if v[:1] != s {
				switch s {
				case "def":
				default:
					t.Fatalf("%s : %s", v, s)
				}
			}
		default:
			t.Fatal("No out")
		}
	}
	time.Sleep(1e7)
}

func TestStar(t *testing.T) {
	out := make(chan string, 1)
	h := make(map[string]http.Handler, 5)
	for _, v := range strings.Split("12345", "") {
		v := v
		h[v] = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			out <- v
		})
	}
	d := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out <- "def"
	})

	c := NewCacheHandler(d, h, nil)
	f, err := os.OpenFile("test_data.txt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	c.AutoSaveTo(f)
	for k, v := range h {
		c.Set("*"+k, k, v)
	}
	req := &http.Request{}
	for _, v := range strings.Split("x0;x1;x2;x3;x4;x5;x6;0;1;2;3;4;5;6", ";") {
		req.Host = v
		c.ServeHTTP(nil, req)
		select {
		case s := <-out:
			if v[len(v)-1:] != s {
				switch s {
				case "def":
				default:
					t.Fatalf("%s : %s", v, s)
				}
			}
		default:
			t.Fatal("No out")
		}
	}
	var obuf bytes.Buffer
	if _, err := c.Save(&obuf); err != nil {
		t.Fatal(err)
	}

	nc := NewCacheHandler(d, h, &obuf)
	for _, v := range strings.Split("x0;x1;x2;x3;x4;x5;x6;0;1;2;3;4;5;6", ";") {
		req.Host = v
		nc.ServeHTTP(nil, req)
		select {
		case s := <-out:
			if v[len(v)-1:] != s {
				switch s {
				case "def":
				default:
					t.Fatalf("%s : %s", v, s)
				}
			}
		default:
			t.Fatal("No out")
		}
	}
	time.Sleep(1e7)
}

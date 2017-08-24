package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/cznic/b"
)

type item struct {
	name string
	h    http.Handler
}

// SyncWriter is interface used for writing cache to file.
type SyncWriter interface {
	io.Writer
	Sync() error
	Seek(offset int64, whence int) (ret int64, err error)
	Truncate(size int64) error
}

// CacheHandler stores a mapping of host to http.Handler in a tree structure.
// CacheHandler implements http.Handler, so it can be used in http.ListenAndServe.
//
// If the request is for local server, it will call the Local handler.
// Otherwise it will search in the tree for handler, or fallback to Default.
//
// The compare function will only compare the request.Host . Wild card can be
// used, but only accepts when it is at the beginning.
type CacheHandler struct {
	tree *b.Tree
	lock sync.RWMutex
	// The default handler when entity is not found in cache.
	// If it is nil, and handler not found, it will response an internal error
	Default http.Handler
	// Handler to handle request to local server. If it is nil,
	// http.DefaultServeMux will be called.
	Local http.Handler

	sLock  sync.Mutex
	writeQ chan struct{}
}

// Compare from the end of string to the beginning.
// Accept wild card at the beginning
func compareHost(a, b string) int {
	var i int
	an, bn := len(a), len(b)
	var as, bs byte
	for i = 1; i <= an && i <= bn; i++ {
		as, bs = a[an-i], b[bn-i]
		if as != bs {
			switch {
			case i == an && as == '*', i == bn && bs == '*':
				return 0
			case as > bs:
				return +1
			default:
				return -1
			}
		}
	}
	return 0
}

// NewCacheHandler return a new CacheHandler with h as default handler.
// If r is not nil, CacheHandler will read r and lookup the handler by hmap.
func NewCacheHandler(h http.Handler, hmap map[string]http.Handler, r io.Reader) *CacheHandler {
	c := &CacheHandler{
		tree: b.TreeNew(func(a, b interface{}) int {
			as, bs := a.(string), b.(string)
			return compareHost(as, bs)
		}),
		Default: h,
	}
	if r != nil {
		c.Read(r, hmap)
	}
	return c
}

// ServeHTTP implements http.Handler
func (c *CacheHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Not a proxy request
	if r.URL.Host == "" {
		h := c.Local
		if h == nil {
			h = http.DefaultServeMux
		}
		h.ServeHTTP(w, r)
		return
	}

	var h http.Handler
	c.lock.RLock()
	e, ok := c.tree.Seek(r.Host)
	if ok {
		_, v, _ := e.Next()
		h = v.(*item).h
		e.Close()
	}
	c.lock.RUnlock()

	if h != nil {
		h.ServeHTTP(w, r)
		return
	}

	if c.Default == nil {
		http.Error(w, "No handler", http.StatusInternalServerError)
		return
	}
	c.Default.ServeHTTP(w, r)
}

// Set stores the mapping of addr to h into the cache.
// It will also write to file if AutoSaveTo is set, and name is not empty.
func (c *CacheHandler) Set(addr, name string, h http.Handler) {
	c.lock.Lock()
	c.set(addr, name, h)
	c.lock.Unlock()

	if name != "" {
		c.sLock.Lock()
		defer c.sLock.Unlock()

		if c.writeQ != nil {
			//non-blocking push
			select {
			case c.writeQ <- struct{}{}:
			default:
			}
		}
	}
}

// set stores the mapping of addr to h into the cache.
// Caller must lock the c.lock before calling this function
func (c *CacheHandler) set(addr, name string, h http.Handler) {
	c.tree.Set(addr, &item{name: name, h: h})
}

// AutoSaveTo set a writer so whenever the cache is updated, CacheHandler will write to it.
// Calling it will remove the previous set writer. Set w to nil to stop auto save.
func (c *CacheHandler) AutoSaveTo(w SyncWriter) {
	c.sLock.Lock()
	defer c.sLock.Unlock()

	if c.writeQ != nil {
		close(c.writeQ)
	}
	if w == nil {
		return
	}
	c.writeQ = make(chan struct{}, 1)
	go func() {
		for range c.writeQ {
			w.Seek(0, 0)
			if n, err := c.Save(w); err == nil {
				w.Sync()
				w.Truncate(int64(n))
			}
		}
	}()
}

// Save writes the cache to w. As handler itself cannot be written, it will write its name.
// Handler without name will not be written.
func (c *CacheHandler) Save(w io.Writer) (n int, err error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	e, err := c.tree.SeekFirst()
	if err != nil {
		if err == io.EOF {
			return 0, nil
		}
		return 0, err
	}
	defer e.Close()

	for err == nil {
		var k, v interface{}
		k, v, err = e.Next()
		if err != nil {
			break
		}
		name := v.(*item).name
		if name != "" {
			nn, err := fmt.Fprintf(w, "%s\t%s\n", name, k.(string))
			n += nn
			if err != nil {
				return n, err
			}
		}
	}

	if err == io.EOF {
		err = nil
	}
	return n, err
}

// Read reads from r, and use hmap to lookup the handler for the hosts
func (c *CacheHandler) Read(r io.Reader, hmap map[string]http.Handler) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	scr := bufio.NewScanner(r)
	for scr.Scan() {
		t := strings.Split(scr.Text(), "\t")
		if len(t) != 2 {
			log.Print("Failed to parse line: " + scr.Text())
			continue
		}
		if h, ok := hmap[t[0]]; ok {
			c.set(t[1], t[0], h)
		} else if h, ok := hmap[t[1]]; ok {
			c.set(t[0], t[1], h)
		} else {
			log.Print("Failed to find the handler: " + t[0])
		}
	}
	return scr.Err()
}

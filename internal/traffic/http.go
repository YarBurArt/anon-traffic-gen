package traffic

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

// TODO: uTLS with a Chrome ClientHello, Go's handshake marks the client.
func newTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 64
	t.MaxIdleConnsPerHost = 4
	t.IdleConnTimeout = 90 * time.Second
	return t
}

// TODO: headers, Referer and cookies per UA, a lone UA reads as a script.
func (g *Generator) get(ctx context.Context, target Target, ua string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.URL, nil)
	if err != nil {
		log.Printf("bad url %q: %v", target.URL, err)
		return
	}
	// the value slice is built once per user agent; Set would canonicalise and
	// allocate a map entry on every request
	req.Header["User-Agent"] = g.agent[ua]
	if g.opt.Debug {
		log.Printf("GET %s as %.40s", target.URL, ua)
	}

	start := time.Now()
	resp, err := g.client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		fmt.Fprintf(g.out, "Request: GET %s\nError: %v\n", target.URL, err)
		return
	}
	// the body is read here, so one fetch gives both the status line and the
	// links to follow
	body, readErr := g.readPage(resp.Body)
	resp.Body.Close()

	fmt.Fprintf(g.out, "Request: GET %s\nResponse: Status Code: %d, Elapsed Time: %s\n",
		target.URL, resp.StatusCode, elapsed)

	if readErr != nil || resp.StatusCode >= http.StatusBadRequest {
		return
	}
	g.absorb(target.base, body)
}

func (g *Generator) readPage(r io.Reader) ([]byte, error) {
	// one buffer for the whole run; a chunk array per call would escape to the
	// heap, because Read takes an interface
	body := g.body[:0]
	if cap(body) < readChunk {
		body = make([]byte, 0, readChunk)
	}

	for len(body) < maxBody {
		have := len(body)
		if cap(body)-have < readChunk {
			grown := make([]byte, have, have+readChunk)
			copy(grown, body)
			body = grown
		}

		n, err := r.Read(body[have:cap(body)])
		body = body[:have+n]
		if err == io.EOF {
			break
		}
		if err != nil {
			g.body = body
			return body, err
		}
		if n == 0 {
			break
		}
	}

	g.body = body
	return body, nil
}

func (g *Generator) absorb(base *url.URL, body []byte) {
	if base == nil {
		return
	}
	g.links = extractLinks(g.links[:0], base, body)
	for i := range g.links {
		link := &g.links[i]
		// loopback links are dropped, so the crawl cannot turn on this machine
		if isLocal(link) {
			continue
		}
		if g.add(link) && g.opt.Debug {
			log.Printf("added url %s", link.String())
		}
	}
}

func isLocal(u *url.URL) bool {
	if u == nil {
		return true
	}
	host := u.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsUnspecified())
}

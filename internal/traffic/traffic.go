// Package traffic generates network noise from the config.
package traffic

import (
	"bufio"
	"context"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/YarBurArt/anon-traffic-gen/internal/config"
)

const (
	// a run lasts for hours, so one page must not decide its footprint
	maxBody  = 256 << 10
	maxLinks = 64

	// without a cap a site that links to itself grows the pool forever
	maxTargets = 512

	// requests go one at a time, a stalled host would hold up everything after it
	defaultTimeout = 30 * time.Second

	readChunk = 4096
	outBuf    = 32 << 10
)

type Target struct {
	URL  string
	WS   string
	base *url.URL // kept so a fetch does not parse the page url again
}

type Options struct {
	Debug      bool
	RWConf     bool
	Torrent    bool
	ConfigPath string
	Out        io.Writer
}

type Generator struct {
	cfg    *config.Config
	opt    Options
	out    *bufio.Writer
	client *http.Client
	dialer *websocket.Dialer

	targets []Target
	seen    map[string]struct{}
	agent   map[string][]string
	links   []url.URL
	body    []byte
	msg     []byte
	dirty   bool
}

func New(cfg *config.Config, opt Options) *Generator {
	if opt.Out == nil {
		opt.Out = io.Discard
	}
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	g := &Generator{
		cfg: cfg,
		opt: opt,
		out: bufio.NewWriterSize(opt.Out, outBuf),
		client: &http.Client{
			Timeout:   timeout,
			Transport: newTransport(),
		},
		dialer: &websocket.Dialer{
			HandshakeTimeout: time.Duration(cfg.WebSocketTimeout) * time.Second,
		},
		seen:  make(map[string]struct{}, len(cfg.URLs)),
		agent: make(map[string][]string, len(cfg.UserAgents)),
	}
	for _, ua := range cfg.UserAgents {
		g.agent[ua] = []string{ua}
	}

	for _, raw := range cfg.URLs {
		u, err := url.Parse(raw)
		if err != nil {
			log.Printf("skipping url %q: %v", raw, err)
			continue
		}
		g.add(u)
	}
	return g
}

// Run generates until the context is cancelled.
func (g *Generator) Run(ctx context.Context) {
	var wg sync.WaitGroup
	if g.opt.Torrent && g.cfg.TorrentLink != "" {
		dir, err := os.MkdirTemp("", "anon-traffic-gen-")
		if err != nil {
			log.Printf("torrent off, no temp dir: %v", err)
		} else {
			defer os.RemoveAll(dir)
			wg.Add(1)
			go func() {
				// a torrent spends its time waiting on peers
				defer wg.Done()
				g.torrent(ctx, dir)
			}()
		}
	}

	for ctx.Err() == nil {
		g.Cycle(ctx)
		g.pause(ctx)
	}
	wg.Wait()
	g.out.Flush()
	if g.opt.RWConf && g.dirty {
		g.save()
	}
}

// Cycle makes one pass over the pool.
//
// TODO: pace each host, one sleep per pass hits everything at one rate.
func (g *Generator) Cycle(ctx context.Context) {
	// taken once, so pages found in this pass wait for the next one
	n := len(g.targets)
	for i := 0; i < n && ctx.Err() == nil; i++ {
		target := g.targets[i]
		for _, ua := range g.cfg.UserAgents {
			g.get(ctx, target, ua)
		}
	}

	if rand.IntN(5) == 0 {
		g.socket(ctx)
	}

	g.out.Flush()
	// rewriting the file on every pass would hammer the disk for nothing
	if g.opt.RWConf && g.dirty {
		g.save()
	}
}

func (g *Generator) pause(ctx context.Context) {
	if g.cfg.RateLimit <= 0 {
		return
	}
	timer := time.NewTimer(time.Duration(g.cfg.RateLimit) * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func (g *Generator) add(u *url.URL) bool {
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	raw := u.String()
	if _, ok := g.seen[raw]; ok {
		return false
	}
	if len(g.targets) >= maxTargets {
		return false
	}

	// the url outlives the page buffer it was parsed from, and the websocket
	// twin needs another scheme, so both get a copy
	own := *u
	twin := own
	g.seen[raw] = struct{}{}
	g.dirty = true
	g.targets = append(g.targets, Target{URL: raw, WS: socketURL(&twin), base: &own})
	return true
}

func (g *Generator) save() {
	g.dirty = false
	g.cfg.URLs = g.cfg.URLs[:0]
	for _, target := range g.targets {
		g.cfg.URLs = append(g.cfg.URLs, target.URL)
	}
	if err := g.cfg.Save(g.opt.ConfigPath); err != nil {
		log.Printf("cannot rewrite %s: %v", g.opt.ConfigPath, err)
	}
}

func socketURL(u *url.URL) string {
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return ""
	}
	return u.String()
}

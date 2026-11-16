// spoof-http fills a segment with ordinary-looking traffic, so that whoever
// watches the network has to pick the real events out of the noise.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/YarBurArt/anon-traffic-gen/internal/config"
	"github.com/YarBurArt/anon-traffic-gen/internal/traffic"
)

func main() {
	flag.CommandLine.Init("spoof-http", flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)

	path := flag.String("config", "config.yaml", "config file")
	stand := flag.String("stand-url", "", "crawl one host, e.g. http://10.0.0.1:8001")
	torrent := flag.Bool("torrent", false, "add torrent chatter")
	debug := flag.Bool("debug", false, "log every request and crawled url")
	rwconf := flag.Bool("rwconf", false, "write crawled urls back to the config")

	flag.StringVar(path, "c", "config.yaml", "shorthand for --config")
	flag.StringVar(stand, "s", "", "shorthand for --stand-url")
	flag.BoolVar(torrent, "t", false, "shorthand for --torrent")
	flag.BoolVar(debug, "d", false, "shorthand for --debug")
	flag.BoolVar(rwconf, "w", false, "shorthand for --rwconf")
	flag.Usage = usage

	// an unknown flag, an unknown argument and help all mean the caller wanted
	// the help text, so print it instead of a complaint
	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		// the flag package printed the usage itself on the way out
		return
	}
	if flag.NArg() > 0 {
		usage()
		return
	}

	cfg, err := config.Load(*path)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	opt := traffic.Options{
		Debug:      *debug,
		RWConf:     *rwconf,
		Torrent:    *torrent,
		ConfigPath: *path,
		Out:        os.Stdout,
	}

	if *stand != "" {
		u, err := url.Parse(*stand)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			log.Fatalf("--stand-url wants a full address like http://10.0.0.1:8001, got %q", *stand)
		}
		opt.Stand = u
		if *rwconf {
			log.Print("--rwconf does nothing on a stand, the pool is rebuilt from it every run")
		}
	}
	if opt.Stand == nil && len(cfg.URLs) == 0 {
		log.Fatal("no urls in the config and no --stand-url")
	}
	if *torrent && cfg.TorrentLink == "" {
		log.Fatal("--torrent needs torrent_link in the config")
	}

	if *debug {
		log.Printf("loaded %s: %d urls, %d user agents", *path, len(cfg.URLs), len(cfg.UserAgents))
	}

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGTSTP)
	defer stop()

	fmt.Println("generating traffic, Ctrl+C to stop")
	traffic.New(cfg, opt).Run(ctx)
}

func usage() {
	fmt.Fprint(os.Stdout, `Generate HTTP, websocket and torrent noise for traffic-analysis labs.

usage:
  spoof-http [flags]

flags:
  --config, -c    path to the yaml config (default "config.yaml")
  --stand-url, -s crawl one host, e.g. http://10.0.0.1:8001
  --torrent, -t   add torrent chatter, off unless asked for
  --debug, -d     log every request and crawled url
  --rwconf, -w    write crawled urls back into the config file

examples:
  noise against the urls in the config:
    spoof-http --config config.yaml
  noise against one lab stand:
    spoof-http --config lab.yaml --stand-url http://10.0.0.1:8001
  a stand plus torrent chatter:
    spoof-http -s http://10.0.0.1:8001 -t
`)
}

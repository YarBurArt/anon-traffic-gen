// spoof-http fills a segment with ordinary-looking traffic, so that whoever
// watches the network has to pick the real events out of the noise.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/YarBurArt/anon-traffic-gen/internal/config"
	"github.com/YarBurArt/anon-traffic-gen/internal/traffic"
)

func main() {
	path := flag.String("config", "config.yaml", "config file")
	debug := flag.Bool("debug", false, "log every request and crawled url")
	rwconf := flag.Bool("rwconf", false, "write crawled urls back to the config")

	flag.Usage = usage
	flag.Parse()

	cfg, err := config.Load(*path)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	opt := traffic.Options{
		Debug:      *debug,
		RWConf:     *rwconf,
		ConfigPath: *path,
		Out:        os.Stdout,
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
  --config    path to the yaml config (default "config.yaml")
  --debug     log every request and crawled url
  --rwconf    write crawled urls back into the config file
`)
}

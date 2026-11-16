package traffic

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/anacrolix/torrent"
)

const torrentPause = 5 * time.Second

// every attempt finds a different set of peers, so the same torrent is pulled
// over and over.
//
// TODO: reuse one client and seed, new clients burst DHT joins.
func (g *Generator) torrent(ctx context.Context, dir string) {
	attempts := g.cfg.MaxRetries
	for i := 0; i < attempts && ctx.Err() == nil; i++ {
		if err := g.pull(ctx, dir); err != nil && ctx.Err() == nil {
			log.Printf("torrent attempt %d: %v", i+1, err)
		}
		if i < attempts-1 {
			select {
			case <-ctx.Done():
			case <-time.After(torrentPause):
			}
		}
	}
}

func (g *Generator) pull(ctx context.Context, dir string) error {
	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = dir
	cfg.Seed = false

	client, err := torrent.NewClient(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	t, err := client.AddMagnet(g.cfg.TorrentLink)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.GotInfo():
	}

	t.DownloadAll()
	done := make(chan struct{})
	go func() {
		client.WaitAll()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}

	// the payload is not wanted, only the connections it took
	for _, f := range t.Files() {
		os.Remove(filepath.Join(dir, f.Path()))
	}
	return nil
}

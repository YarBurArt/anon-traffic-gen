package traffic

import (
	"context"
	"log"
	"math/rand/v2"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

// TODO: hold the socket open and space frames out, dial-close is a tell.
func (g *Generator) socket(ctx context.Context) {
	for i := 0; i < len(g.targets) && ctx.Err() == nil; i++ {
		target := g.targets[i]
		if target.WS == "" {
			continue
		}
		for _, ua := range g.cfg.UserAgents {
			header := http.Header{}
			header.Set("User-Agent", ua)

			conn, _, err := g.dialer.DialContext(ctx, target.WS, header)
			if err != nil {
				if g.opt.Debug {
					log.Printf("websocket %s: %v", target.WS, err)
				}
				continue
			}
			// the handshake carries the noise, one message is enough
			conn.WriteMessage(websocket.TextMessage, g.message())
			conn.Close()
		}
	}
}

func (g *Generator) message() []byte {
	g.msg = append(g.msg[:0], "WS message: "...)
	return strconv.AppendInt(g.msg, int64(rand.IntN(1000)), 10)
}

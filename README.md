# anon-traffic-gen

Background traffic for IR labs. The generator walks a pool of urls,
fetches each page with several user agents, follows the links it finds, dials
websockets every so often, and adds torrent chatter if you ask for it. Enough
going on that a blue team has something to sort the real events out of.

Check the result with the tools that will be doing the watching:
[Arkime](https://github.com/arkime/arkime) for the session view,
[nDPI](https://github.com/ntop/ndpi) for the protocol parsing. If either one can
pick the generated traffic out of a capture at a glance, the noise needs more
work.

## usage
```
spoof-http [flags]

    --config, -c    path to the yaml config (default "config.yaml")
    --stand-url, -s crawl one host, e.g. http://10.0.0.1:8001
    --torrent, -t   add torrent chatter, off unless asked for
    --debug, -d     log every request and crawled url
    --rwconf, -w    write crawled urls back into the config file
```

Torrent chatter is opt-in: without `-t` the client never starts and the magnet
link in the config is ignored.

## stand mode

A lab target usually means one address, so point the generator at it:

```bash
spoof-http --config=lab.yaml --stand-url http://10.0.0.1:8001
```

The pool starts as that address and the crawl stays on the same host, so every
page the stand links to is fetched with every user agent, pass after pass. The
pool is rebuilt from the stand on each run, which is why `--rwconf` does nothing
in this mode. A stand on loopback works too.

## config

```yaml
urls:
  - https://example.com
user_agents:
  - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"
rate_limit: 0        # seconds between passes, 0 means no pause
timeout: 30          # per request, seconds
websocket_timeout: 10
max_retries: 3       # torrent attempts, only with --torrent
torrent_link: "magnet:?xt=urn:btih:..."
```

Unknown keys are an error, so a typo fails at startup instead of doing nothing.
The pool is deduplicated and capped at 512 urls, because a page that links to
itself would otherwise grow it on every pass. Crawled urls stay in memory unless
`--rwconf` writes them back to the file.

## build

```bash
go build -o spoof-http ./cmd/spoof-http
```

```bash
docker build -t anon-traffic-gen .
```

```bash 
docker run --rm anon-traffic-gen
```

The image is a static binary on alpine plus ca-certificates, built with
`CGO_ENABLED=0`. Locally cgo is optional.

## Dependencies

- [anacrolix/torrent](https://github.com/anacrolix/torrent) for peer traffic and DHT joins
- [gorilla/websocket](https://github.com/gorilla/websocket) for the websocket dials
- [yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) for the config file

## Contributing

If you find any issues or have suggestions for improvements, feel free to open an issue or submit a pull request on the [anon-traffic-gen](https://github.com/YarBurArt/anon-traffic-gen/tree/main).

## License

This project is licensed under the [MIT License](https://github.com/YarBurArt/anon-traffic-gen/blob/main/LICENSE).


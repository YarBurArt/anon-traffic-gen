package traffic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/YarBurArt/anon-traffic-gen/internal/config"
)

const testPage = `<!doctype html><html><body>
<a href="/one">one</a><a href="two">two</a><a href="#skip">skip</a>
<a href="javascript:void(0)">js</a><a href="https://example.test/three">three</a>
</body></html>`

// an absolute link would send later cycles at a host that does not exist
const benchPage = `<!doctype html><html><body>
<a href="/one">one</a><a href="/two">two</a><a href="/three">three</a>
</body></html>`

func pageServer(tb testing.TB, page string) *httptest.Server {
	tb.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, page)
	}))
	tb.Cleanup(srv.Close)
	return srv
}

func socketServer(tb testing.TB) *httptest.Server {
	tb.Helper()
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		conn.ReadMessage()
		conn.Close()
	}))
	tb.Cleanup(srv.Close)
	return srv
}

func testConfig(urls ...string) *config.Config {
	return &config.Config{
		URLs:             urls,
		UserAgents:       []string{"ua-a", "ua-b"},
		Timeout:          5,
		WebSocketTimeout: 2,
	}
}

func TestCycleRequestsOncePerUserAgent(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "" {
			return // the websocket pass lands here too, it is not a page fetch
		}
		hits.Add(1)
		if got := r.Header.Get("User-Agent"); !strings.HasPrefix(got, "ua-") {
			t.Errorf("user agent %q did not come from the config", got)
		}
		fmt.Fprint(w, testPage)
	}))
	defer srv.Close()

	var out strings.Builder
	g := New(testConfig(srv.URL), Options{Out: &out})
	g.Cycle(context.Background())

	if got := hits.Load(); got != 2 {
		t.Fatalf("requests: got %d, want one per user agent (2)", got)
	}
	if !strings.Contains(out.String(), "Status Code: 200") {
		t.Fatalf("report missing status line:\n%s", out.String())
	}
}

func TestAbsorbResolvesAndSkipsJunk(t *testing.T) {
	g := New(testConfig("http://10.0.0.1/"), Options{})
	g.absorb(mustParse(t, "https://news.test/index.html"), []byte(testPage))

	want := []string{
		"https://news.test/one",
		"https://news.test/two",
		"https://example.test/three",
	}
	if len(g.targets) != len(want)+1 {
		t.Fatalf("pool: got %d entries, want %d", len(g.targets), len(want)+1)
	}
	for i, url := range want {
		if got := g.targets[i+1].URL; got != url {
			t.Errorf("link %d: got %q, want %q", i, got, url)
		}
	}
}

func TestAbsorbDropsLoopback(t *testing.T) {
	g := New(testConfig("http://10.0.0.1/"), Options{})
	g.absorb(mustParse(t, "http://127.0.0.1:8080/index.html"), []byte(`<a href="/one">a</a><a href="two">b</a>`))

	if len(g.targets) != 1 {
		t.Fatalf("loopback links reached the pool: %v", g.targets)
	}
}

func TestAddKeepsPoolBoundedAndUnique(t *testing.T) {
	g := New(testConfig("https://site.test/"), Options{})
	if !g.add(mustParse(t, "https://site.test/a")) {
		t.Fatal("new url was not added")
	}
	if g.add(mustParse(t, "https://site.test/a")) {
		t.Fatal("same url was added twice")
	}
	if g.add(mustParse(t, "ftp://site.test/file")) {
		t.Fatal("non-http url was added")
	}

	g.targets = make([]Target, maxTargets)
	for i := 0; i < 10; i++ {
		if g.add(mustParse(t, fmt.Sprintf("https://site.test/cap-%d", i))) {
			t.Fatal("pool grew past maxTargets")
		}
	}
}

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestExtractLinks(t *testing.T) {
	base := mustParse(t, "http://site.test/dir/page.html")
	body := []byte(`<a href="one">a</a><A HREF='/two'>b</A><a href="#frag">c</a>
<a href="mailto:x@y">d</a><a href="//other.test/path">e</a><a>f</a>
<a href="/three" class="x">g</a><img src="pic.png">`)

	got := extractLinks(nil, base, body)
	want := []string{
		"http://site.test/dir/one",
		"http://site.test/two",
		"http://other.test/path",
		"http://site.test/three",
	}
	if len(got) != len(want) {
		t.Fatalf("links: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i].String() != want[i] {
			t.Fatalf("link %d: got %q, want %q", i, got[i].String(), want[i])
		}
	}
}

func TestStandModeKeepsTheCrawlOnTheStand(t *testing.T) {
	stand := mustParse(t, "http://127.0.0.1:8001/")
	g := New(testConfig(), Options{Stand: stand})

	if len(g.targets) != 1 || g.targets[0].URL != "http://127.0.0.1:8001/" {
		t.Fatalf("pool did not start from the stand: %v", g.targets)
	}
	if g.stand != "127.0.0.1:8001" {
		t.Fatalf("stand host: got %q", g.stand)
	}

	g.absorb(stand, []byte(`<a href="/a">a</a><a href="http://other.test/b">b</a>`))
	if len(g.targets) != 2 {
		t.Fatalf("pool: got %d entries, want the stand plus /a: %v", len(g.targets), g.targets)
	}
	if got := g.targets[1].URL; got != "http://127.0.0.1:8001/a" {
		t.Fatalf("same-host link: got %q", got)
	}
}

func BenchmarkGet(b *testing.B) {
	srv := pageServer(b, benchPage)
	g := New(testConfig(srv.URL), Options{Out: io.Discard})
	target := g.targets[0]
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.get(ctx, target, "ua-a")
	}
}

func BenchmarkCycle(b *testing.B) {
	srv := pageServer(b, benchPage)
	g := New(testConfig(srv.URL), Options{Out: io.Discard})
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Cycle(ctx)
	}
}

func BenchmarkSocket(b *testing.B) {
	srv := socketServer(b)
	g := New(testConfig(srv.URL), Options{Out: io.Discard})
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.socket(ctx)
	}
}

func BenchmarkExtractLinks(b *testing.B) {
	base, err := url.Parse("https://site.test/index.html")
	if err != nil {
		b.Fatal(err)
	}
	body := []byte(testPage)
	buf := make([]url.URL, 0, maxLinks)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf = extractLinks(buf[:0], base, body)
	}
}

package traffic

import (
	"bytes"
	"net/url"
	"strings"
)

var (
	hrefAttr    = []byte("href")
	commentOpen = []byte("<!--")
	commentEnd  = []byte("-->")
	scriptName  = []byte("script")
	scriptClose = []byte("</script")
	styleName   = []byte("style")
	styleClose  = []byte("</style")
)

// extractLinks appends the links of a page to dst.
func extractLinks(dst []url.URL, base *url.URL, body []byte) []url.URL {
	// the page is only a source of urls, so the bytes are scanned for <a href>
	// instead of parsed: a tokenizer allocates a buffer per page and rewrites
	// every chunk it reads
	for i := 0; i < len(body) && len(dst) < maxLinks; i++ {
		if body[i] != '<' {
			continue
		}
		rest := body[i:]

		// javascript and css are full of href-looking strings, and a link inside
		// a comment was never offered to a visitor
		if hasPrefixFold(rest, commentOpen) {
			end := bytes.Index(rest, commentEnd)
			if end < 0 {
				return dst
			}
			i += end + len(commentEnd) - 1
			continue
		}
		if closer, ok := blockClose(rest); ok {
			end := indexFold(rest, closer)
			if end < 0 {
				return dst
			}
			i += end + len(closer) - 1
			continue
		}

		tag := rest[1:]
		if len(tag) < 2 || (tag[0] != 'a' && tag[0] != 'A') {
			continue
		}
		if c := tag[1]; c != ' ' && c != '\t' && c != '\n' && c != '\r' && c != '>' && c != '/' {
			continue
		}
		end := bytes.IndexByte(tag, '>')
		if end < 0 {
			return dst
		}
		if raw, ok := attrValue(tag[:end], hrefAttr); ok {
			if u, ok := resolveLink(base, raw); ok {
				dst = append(dst, *u)
			}
		}
		i += end
	}
	return dst
}

func blockClose(tag []byte) ([]byte, bool) {
	switch {
	case opensWithName(tag, scriptName):
		return scriptClose, true
	case opensWithName(tag, styleName):
		return styleClose, true
	}
	return nil, false
}

func opensWithName(tag, name []byte) bool {
	if len(tag) < len(name)+1 || !bytes.EqualFold(tag[1:len(name)+1], name) {
		return false
	}
	if len(tag) == len(name)+1 {
		return true
	}
	// the name has to end here, <stylesheet> is not a style block
	switch tag[len(name)+1] {
	case ' ', '\t', '\n', '\r', '>', '/':
		return true
	}
	return false
}

func hasPrefixFold(b, prefix []byte) bool {
	return len(b) >= len(prefix) && bytes.EqualFold(b[:len(prefix)], prefix)
}

func indexFold(haystack, needle []byte) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if bytes.EqualFold(haystack[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}

func attrValue(tag, name []byte) (string, bool) {
	for i := 0; i < len(tag); {
		for i < len(tag) && (isSpace(tag[i]) || tag[i] == '/') {
			i++
		}

		start := i
		for i < len(tag) && !isSpace(tag[i]) && tag[i] != '=' {
			i++
		}
		key := tag[start:i]
		for i < len(tag) && isSpace(tag[i]) {
			i++
		}
		if i >= len(tag) || tag[i] != '=' {
			continue
		}

		i++
		for i < len(tag) && isSpace(tag[i]) {
			i++
		}

		start = i
		if i < len(tag) && (tag[i] == '"' || tag[i] == '\'') {
			quote := tag[i]
			i++
			start = i
			for i < len(tag) && tag[i] != quote {
				i++
			}
			value := string(tag[start:i])
			if i < len(tag) {
				i++
			}
			if bytes.EqualFold(key, name) {
				return value, true
			}
			continue
		}

		for i < len(tag) && !isSpace(tag[i]) {
			i++
		}
		if bytes.EqualFold(key, name) {
			return string(tag[start:i]), true
		}
	}
	return "", false
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

func resolveLink(base *url.URL, raw string) (*url.URL, bool) {
	if raw == "" || raw[0] == '#' {
		return nil, false
	}
	if strings.HasPrefix(raw, "javascript:") || strings.HasPrefix(raw, "mailto:") ||
		strings.HasPrefix(raw, "tel:") || strings.HasPrefix(raw, "data:") {
		return nil, false
	}

	ref, err := url.Parse(raw)
	if err != nil {
		return nil, false
	}
	u := base.ResolveReference(ref)
	if u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, false
	}
	u.Fragment = ""
	return u, true
}

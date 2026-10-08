package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
)

// pageReader fetches a shared page from the internet, never from this
// computer itself: a link to localhost would have Sameway read what only
// this computer should see, for whoever shared it.
var pageReader = &http.Client{
	Timeout: 20 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{Timeout: 10 * time.Second, Control: func(_, address string, _ syscall.RawConn) error {
			host, _, _ := net.SplitHostPort(address)
			if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast()) && !readLocal {
				return errors.New("a page on this computer is not read")
			}
			return nil
		}}).DialContext,
	},
}

// readLocal lets a test read a page it serves on this computer.
var readLocal = false

// pageMost is how much of a page is read, and how many characters of its
// words are kept: a long article whole, a book not.
const (
	pageMost  = 5 << 20
	wordsMost = 40000
)

// readPage is a web page's title and its main words as Markdown: the
// article, or the main part, or the body, without its menus, headers,
// footers, scripts and forms.
func readPage(ctx context.Context, link string) (title, words string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Sameway; saving a page someone shared)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	res, err := pageReader.Do(req)
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return "", "", errors.New("it took too long to answer")
		}
		if strings.Contains(err.Error(), "not read") {
			return "", "", errors.New("a page on this computer is not read")
		}
		return "", "", errors.New("it could not be reached")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("the site answered %s", strings.ToLower(http.StatusText(res.StatusCode)))
	}
	if kind, _, _ := mime.ParseMediaType(res.Header.Get("Content-Type")); kind != "text/html" && kind != "application/xhtml+xml" {
		return "", "", errors.New("it is not a web page")
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, pageMost))
	if err != nil {
		return "", "", errors.New("it did not arrive whole")
	}
	doc, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		return "", "", errors.New("its words could not be read")
	}
	title = oneLineOf(wordsOf(firstOf(doc, atom.Title)))
	if og := metaContent(doc, "og:title"); og != "" {
		title = og
	}
	main := firstOf(doc, atom.Article)
	if main == nil {
		main = firstOf(doc, atom.Main)
	}
	if main == nil {
		main = firstOf(doc, atom.Body)
	}
	if main == nil {
		return title, "", nil
	}
	prune(main)
	var b bytes.Buffer
	html.Render(&b, main)
	md, err := convert.HTMLToMarkdown(b.String())
	if err != nil {
		return title, "", nil
	}
	return title, clipRunes(strings.TrimSpace(md), wordsMost), nil
}

// firstOf is the first element of a kind, in document order.
func firstOf(n *html.Node, a atom.Atom) *html.Node {
	if n.Type == html.ElementNode && n.DataAtom == a {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if f := firstOf(c, a); f != nil {
			return f
		}
	}
	return nil
}

func wordsOf(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(wordsOf(c))
	}
	return b.String()
}

func metaContent(doc *html.Node, property string) string {
	var out string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if out != "" {
			return
		}
		if n.Type == html.ElementNode && n.DataAtom == atom.Meta {
			var prop, content string
			for _, a := range n.Attr {
				switch a.Key {
				case "property", "name":
					prop = a.Val
				case "content":
					content = a.Val
				}
			}
			if prop == property {
				out = oneLineOf(content)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out
}

// noise is what a page has around its words.
var noise = map[atom.Atom]bool{atom.Script: true, atom.Style: true, atom.Noscript: true, atom.Nav: true, atom.Header: true,
	atom.Footer: true, atom.Aside: true, atom.Form: true, atom.Iframe: true, atom.Svg: true, atom.Button: true, atom.Template: true}

func prune(n *html.Node) {
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		if c.Type == html.ElementNode && noise[c.DataAtom] {
			n.RemoveChild(c)
		} else if c.Type == html.CommentNode {
			n.RemoveChild(c)
		} else {
			prune(c)
		}
		c = next
	}
}

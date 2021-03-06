package html

import (
	"bytes"
	"context"
	"fmt"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
	"github.com/shurcooL/highlight_diff"
	"github.com/shurcooL/highlight_go"
	"github.com/shurcooL/octicon"
	"github.com/shurcooL/sanitized_anchor_name"
	"github.com/sourcegraph/annotate"
	"github.com/sourcegraph/syntaxhighlight"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

// Markdown 基本对象
type Markdown struct {
	// cssFile css文件地址
	CssFile []string
	// cssBlock css代码块
	CssBlock string
	JsBlock  string
}

//Html 将markdown转为html
func (markdown *Markdown) Html(content []byte) []byte {
	css := baseCssStyle
	js := ""
	if len(markdown.CssBlock) != 0 {
		css += markdown.CssBlock
	}
	if len(markdown.CssFile) != 0 {

	}
	if len(markdown.JsBlock) != 0 {
		js += markdown.JsBlock
	}

	r := &renderer{Html: blackfriday.HtmlRenderer(0, "", "").(*blackfriday.Html)}
	unsafe := blackfriday.Markdown(content, r, extensions)
	sanitized := policy.SanitizeBytes(unsafe)

	htmlContent := `<html>
<head>
<meta charset="utf-8">
	<link href="https://cdn.bootcdn.net/ajax/libs/github-markdown-css/5.1.0/github-markdown-light.css" media="all" rel="stylesheet" type="text/css" />
	<style>
` + css + `
	</style>
</head>
<body>
<article class="markdown-body entry-content" style="padding: 30px;">`
	htmlContent += string(sanitized)
	htmlContent += `</article>
</body>
<script>
` + js + `
</script>
</html>`
	return []byte(htmlContent)
}

// Pdf 转为pdf
func (markdown *Markdown) Pdf(content []byte) []byte {
	content = markdown.Html(content)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, string(content))
	}))
	defer ts.Close()
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	var pdfByte []byte
	chromedp.Run(ctx,
		chromedp.Navigate(ts.URL),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// to pdf
			buf, _, err := page.PrintToPDF().WithPrintBackground(true).Do(ctx)
			if err != nil {
				return err
			}
			pdfByte = buf
			return nil
		}),
	)
	return pdfByte
}

type renderer struct {
	*blackfriday.Html
}

const baseCssStyle = `pre {
    white-space: pre-wrap !important;
    word-wrap: break-word !important;
}

code {
    white-space: pre-wrap !important;
    word-wrap: break-word !important;
}`

const extensions = blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
	blackfriday.EXTENSION_TABLES |
	blackfriday.EXTENSION_FENCED_CODE |
	blackfriday.EXTENSION_AUTOLINK |
	blackfriday.EXTENSION_STRIKETHROUGH |
	blackfriday.EXTENSION_SPACE_HEADERS |
	blackfriday.EXTENSION_NO_EMPTY_LINE_BEFORE_BLOCK

var policy = func() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").Matching(bluemonday.SpaceSeparatedTokens).OnElements("div", "span")
	p.AllowAttrs("class", "name").Matching(bluemonday.SpaceSeparatedTokens).OnElements("a")
	p.AllowAttrs("rel").Matching(regexp.MustCompile(`^nofollow$`)).OnElements("a")
	p.AllowAttrs("aria-hidden").Matching(regexp.MustCompile(`^true$`)).OnElements("a")
	p.AllowAttrs("type").Matching(regexp.MustCompile(`^checkbox$`)).OnElements("input")
	p.AllowAttrs("checked", "disabled").Matching(regexp.MustCompile(`^$`)).OnElements("input")
	p.AllowDataURIImages()
	return p
}()

func Heading(heading atom.Atom, title string) *html.Node {
	aName := sanitized_anchor_name.Create(title)
	a := &html.Node{
		Type: html.ElementNode, Data: atom.A.String(),
		Attr: []html.Attribute{
			{Key: atom.Name.String(), Val: aName},
			{Key: atom.Class.String(), Val: "anchor"},
			{Key: atom.Href.String(), Val: "#" + aName},
			{Key: atom.Rel.String(), Val: "nofollow"},
			{Key: "aria-hidden", Val: "true"},
		},
	}
	span := &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr:       []html.Attribute{{Key: atom.Class.String(), Val: "octicon-link"}}, // TODO: Factor out the CSS for just headings.
		FirstChild: octicon.Link(),
	}
	a.AppendChild(span)
	h := &html.Node{Type: html.ElementNode, Data: heading.String()}
	h.AppendChild(a)
	h.AppendChild(&html.Node{Type: html.TextNode, Data: title})
	return h
}

func (*renderer) Header(out *bytes.Buffer, text func() bool, level int, _ string) {
	marker := out.Len()
	doubleSpace(out)

	if !text() {
		out.Truncate(marker)
		return
	}

	textHTML := out.String()[marker:]
	out.Truncate(marker)

	// Extract text content of the heading.
	var textContent string
	if node, err := html.Parse(strings.NewReader(textHTML)); err == nil {
		textContent = extractText(node)
	} else {
		// Failed to parse HTML (probably can never happen), so just use the whole thing.
		textContent = html.UnescapeString(textHTML)
	}
	anchorName := sanitized_anchor_name.Create(textContent)

	out.WriteString(fmt.Sprintf(`<h%d><a name="%s" class="anchor" href="#%s" rel="nofollow" aria-hidden="true"><span class="octicon octicon-link"></span></a>`, level, anchorName, anchorName))
	out.WriteString(textHTML)
	out.WriteString(fmt.Sprintf("</h%d>\n", level))
}

func (*renderer) BlockCode(out *bytes.Buffer, text []byte, lang string) {
	doubleSpace(out)

	// parse out the language name
	count := 0
	for _, elt := range strings.Fields(lang) {
		if elt[0] == '.' {
			elt = elt[1:]
		}
		if len(elt) == 0 {
			continue
		}
		out.WriteString(`<div class="highlight highlight-`)
		attrEscape(out, []byte(elt))
		lang = elt
		out.WriteString(`"><pre>`)
		count++
		break
	}

	if count == 0 {
		out.WriteString("<pre><code>")
	}

	if highlightedCode, ok := highlightCode(text, lang); ok {
		out.Write(highlightedCode)
	} else {
		attrEscape(out, text)
	}

	if count == 0 {
		out.WriteString("</code></pre>\n")
	} else {
		out.WriteString("</pre></div>\n")
	}
}

// ListItem Task List support.
func (r *renderer) ListItem(out *bytes.Buffer, text []byte, flags int) {
	switch {
	case bytes.HasPrefix(text, []byte("[ ] ")):
		text = append([]byte(`<input type="checkbox" disabled="">`), text[3:]...)
	case bytes.HasPrefix(text, []byte("[x] ")) || bytes.HasPrefix(text, []byte("[X] ")):
		text = append([]byte(`<input type="checkbox" checked="" disabled="">`), text[3:]...)
	}
	r.Html.ListItem(out, text, flags)
}

var gfmHTMLConfig = syntaxhighlight.HTMLConfig{
	String:        "s",
	Keyword:       "k",
	Comment:       "c",
	Type:          "n",
	Literal:       "o",
	Punctuation:   "p",
	Plaintext:     "n",
	Tag:           "tag",
	HTMLTag:       "htm",
	HTMLAttrName:  "atn",
	HTMLAttrValue: "atv",
	Decimal:       "m",
}

func highlightCode(src []byte, lang string) (highlightedCode []byte, ok bool) {
	switch lang {
	case "Go", "Go-unformatted":
		var buf bytes.Buffer
		err := highlight_go.Print(src, &buf, syntaxhighlight.HTMLPrinter(gfmHTMLConfig))
		if err != nil {
			return nil, false
		}
		return buf.Bytes(), true
	case "diff":
		anns, err := highlight_diff.Annotate(src)
		if err != nil {
			return nil, false
		}

		lines := bytes.Split(src, []byte("\n"))
		lineStarts := make([]int, len(lines))
		var offset int
		for lineIndex := 0; lineIndex < len(lines); lineIndex++ {
			lineStarts[lineIndex] = offset
			offset += len(lines[lineIndex]) + 1
		}

		lastDel, lastIns := -1, -1
		for lineIndex := 0; lineIndex < len(lines); lineIndex++ {
			var lineFirstChar byte
			if len(lines[lineIndex]) > 0 {
				lineFirstChar = lines[lineIndex][0]
			}
			switch lineFirstChar {
			case '+':
				if lastIns == -1 {
					lastIns = lineIndex
				}
			case '-':
				if lastDel == -1 {
					lastDel = lineIndex
				}
			default:
				if lastDel != -1 || lastIns != -1 {
					if lastDel == -1 {
						lastDel = lastIns
					} else if lastIns == -1 {
						lastIns = lineIndex
					}

					beginOffsetLeft := lineStarts[lastDel]
					endOffsetLeft := lineStarts[lastIns]
					beginOffsetRight := lineStarts[lastIns]
					endOffsetRight := lineStarts[lineIndex]

					anns = append(anns, &annotate.Annotation{Start: beginOffsetLeft, End: endOffsetLeft, Left: []byte(`<span class="gd input-block">`), Right: []byte(`</span>`), WantInner: 0})
					anns = append(anns, &annotate.Annotation{Start: beginOffsetRight, End: endOffsetRight, Left: []byte(`<span class="gi input-block">`), Right: []byte(`</span>`), WantInner: 0})

					if '@' != lineFirstChar {
						//leftContent := string(src[beginOffsetLeft:endOffsetLeft])
						//rightContent := string(src[beginOffsetRight:endOffsetRight])
						// This is needed to filter out the "-" and "+" at the beginning of each line from being highlighted.
						// TODO: Still not completely filtered out.
						leftContent := ""
						for line := lastDel; line < lastIns; line++ {
							leftContent += "\x00" + string(lines[line][1:]) + "\n"
						}
						rightContent := ""
						for line := lastIns; line < lineIndex; line++ {
							rightContent += "\x00" + string(lines[line][1:]) + "\n"
						}

						var sectionSegments [2][]*annotate.Annotation
						highlight_diff.HighlightedDiffFunc(leftContent, rightContent, &sectionSegments, [2]int{beginOffsetLeft, beginOffsetRight})

						anns = append(anns, sectionSegments[0]...)
						anns = append(anns, sectionSegments[1]...)
					}
				}
				lastDel, lastIns = -1, -1
			}
		}

		sort.Sort(anns)

		out, err := annotate.Annotate(src, anns, template.HTMLEscape)
		if err != nil {
			return nil, false
		}
		return out, true
	default:
		return nil, false
	}
}

func doubleSpace(out *bytes.Buffer) {
	if out.Len() > 0 {
		out.WriteByte('\n')
	}
}

// extractText returns the recursive concatenation of the text content of an html node.
func extractText(n *html.Node) string {
	var out string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			out += c.Data
		} else {
			out += extractText(c)
		}
	}
	return out
}

func escapeSingleChar(char byte) (string, bool) {
	if char == '"' {
		return "&quot;", true
	}
	if char == '&' {
		return "&amp;", true
	}
	if char == '<' {
		return "&lt;", true
	}
	if char == '>' {
		return "&gt;", true
	}
	return "", false
}

func attrEscape(out *bytes.Buffer, src []byte) {
	org := 0
	for i, ch := range src {
		if entity, ok := escapeSingleChar(ch); ok {
			if i > org {
				// copy all the normal characters since the last escape
				out.Write(src[org:i])
			}
			org = i + 1
			out.WriteString(entity)
		}
	}
	if org < len(src) {
		out.Write(src[org:])
	}
}

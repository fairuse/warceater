package parsers

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/fairuse/warceater/pkg/forum"
	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type GenericParserConfig struct {
	PostSelector string
	UserSelector string
	MsgSelector  string
	HdrSelector  string
}

type GenericForumParser struct {
	// some space for parser specific data here
	Config GenericParserConfig
}

func newGenericForumParser() GenericForumParser {
	return GenericForumParser{Config: GenericParserConfig{
		PostSelector: ".content-border",
		UserSelector: ".post-user",
		MsgSelector:  ".post-message",
		HdrSelector:  ".post-header",
	}}
}

func (fp *GenericForumParser) ParseResponse(body io.Reader, header http.Header, uri string) ([]forum.Post, error) {
	sanitizer := bluemonday.UGCPolicy()

	ctype := header.Get("content-type")
	if !strings.HasPrefix(ctype, "text/html") {
		return nil, fmt.Errorf("not text/html")
	}

	root, err := html.Parse(body)
	if err != nil {
		log.Println("error parsing response body", err)
		return nil, err
	}

	threadUrl, err := url.Parse(uri)
	// we have built some logic that can either get the threadid from a query parameter, or from a part of the threadUrl
	threadIdStr := threadUrl.Query().Get("t") // todo <- make customizable
	// threadId, err := strconv.Atoi(threadIdStr)

	pageSeqStr := threadUrl.Query().Get("page")
	pageSeq, err := strconv.Atoi(pageSeqStr)

	if err != nil {
		// fmt.Println("failed to parse thread identifier for URL", uri)
		// fmt.Println(threadUrl.Query())
		pageSeq = 1
	}
	//fmt.Println(threadUrl.Query())

	doc := goquery.NewDocumentFromNode(root) // not sure where to pass URI.. the internal constructor supports it, but it is not available to us

	bodies := make([]forum.Post, 0)

	doc.Find(fp.Config.PostSelector).Each(func(postNr int, s *goquery.Selection) {
		// For each item found, get the band and title

		//TODO: make a generic struct that allows configured extraction/selecting so:
		// contains a selector (to find) and a means to extract (attribute name, Text, or something else)
		id, _ := s.Attr("id")
		user := s.Find(fp.Config.UserSelector).Text()
		msg := s.Find(fp.Config.MsgSelector).Text()
		hdr := s.Find(fp.Config.HdrSelector).Text()
		userIconUri, _ := s.Find(".post-avatar").Find(".avatar").Find("img").Attr("src")
		// fmt.Println(userIconUri, ok)

		_ = sanitizer
		ohtml, _ := goquery.OuterHtml(s.Find(".post-message")) // todo handle error
		// todo apply transformation rules to modify html (or store the unsanitized html instead, and sanitize on retrieval
		sanehtml := sanitizer.Sanitize(ohtml)
		if len(msg) > 0 {
			// fmt.Printf("Post %s [%d]: %s : %s - %s\n", id, postNr, user, len(hdr), len(msg))
			x := forum.Post{
				Url:          uri,
				ThreadId:     threadIdStr,
				PageSeq:      pageSeq,
				PostSeq:      postNr,
				ThreadPostId: fmt.Sprintf("%012d-%05d", pageSeq, postNr),
				Id:           id,
				User:         user,
				UserIcon:     userIconUri,
				Hdr:          hdr,
				Msg:          msg,
				Html:         sanehtml,
			}
			bodies = append(bodies, x)
		}
	})
	return bodies, nil
}

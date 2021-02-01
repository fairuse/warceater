package parsers

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/microcosm-cc/bluemonday"
	"github.com/fairuse/warceater/pkg/forum"
	"golang.org/x/net/html"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type LeagueForumParser struct {
	// some space for parser specific data here
}

func (fp *LeagueForumParser) ParseResponse(r *http.Response, uri string) ([]forum.Post, error) {
	sanitizer := bluemonday.UGCPolicy()

	ctype := r.Header.Get("content-type")
	if !strings.HasPrefix(ctype, "text/html") {
		return nil, fmt.Errorf("not text/html")
	}

	root, err := html.Parse(r.Body)
	if err != nil {
		log.Println("error parsing response body", err)
		return nil, err
	}

	threadUrl, err := url.Parse(uri)
	// we have built some logic that can either get the threadid from a query parameter, or from a part of the threadUrl
	threadIdStr := threadUrl.Query().Get("t") // todo <- make customizable
	threadId, err := strconv.Atoi(threadIdStr)

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

	doc.Find(".content-border").Each(func(postNr int, s *goquery.Selection) {
		// For each item found, get the band and title
		id, _ := s.Attr("id")
		user := s.Find(".post-user").Text()
		msg := s.Find(".post-message").Text()
		hdr := s.Find(".post-header").Text()
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
				ThreadId:     threadId,
				PageSeq:      pageSeq,
				PostSeq:      postNr,
				ThreadPostId: int64(pageSeq)*1000 + int64(postNr),
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

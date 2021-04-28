package parsers

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/fairuse/warceater/pkg/forum"
	"golang.org/x/net/html"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type YahooAnswersParser struct {
	// some space for parser specific data here
}

// PagePayload is the structure of the embedded json+ld data when an answer page has <10 answers
type PagePayload struct {
	Context    string     `json:"@context"`
	Type       string     `json:"@type"`
	Mainentity Mainentity `json:"mainEntity"`
}
type Author struct {
	Type string `json:"@type"`
	Name string `json:"name"`
} // Sadly, the Author struct as returned in the page-embedded version does not contain unique author IDs

type Answer struct {
	Type        string    `json:"@type"`
	Text        string    `json:"text"`
	Author      Author    `json:"author"`
	Datecreated time.Time `json:"dateCreated"`
	Upvotecount int       `json:"upvoteCount"`
}
type Mainentity struct {
	Type            string    `json:"@type"`
	Name            string    `json:"name"`
	Text            string    `json:"text"`
	Answercount     int       `json:"answerCount"`
	Datecreated     time.Time `json:"dateCreated"`
	Author          Author    `json:"author"`
	Acceptedanswer  Answer    `json:"acceptedAnswer"`
	Suggestedanswer []Answer  `json:"suggestedAnswer"`
}

// ServicePayload is the structure as seen using the PUT requests to the _reservice_ endpoint
type ServicePayload struct {
	Type      string `json:"type"`
	Reservice struct {
		Name           string `json:"name"`
		Start          string `json:"start"`
		State          string `json:"state"`
		PreviousAction struct {
			KvPayload struct {
				Key            string `json:"key"`
				KvActionPrefix string `json:"kvActionPrefix"`
			} `json:"kvPayload"`
			Payload struct {
				Count    int    `json:"count"`
				Lang     string `json:"lang"`
				Qid      string `json:"qid"`
				SortType string `json:"sortType"`
				Start    int    `json:"start"`
			} `json:"payload"`
			Reservice struct {
				Name  string `json:"name"`
				Start string `json:"start"`
				State string `json:"state"`
			} `json:"reservice"`
			Type string `json:"type"`
		} `json:"previous_action"`
	} `json:"reservice"`
	Payload json.RawMessage `json:"payload"`
	Error   bool            `json:"error"`
}

// QAPagePayload is the structure as returned when using the PUT requests to the _reservice_ endpoint
type QAPagePayload struct {
	Qid         string `json:"qid"`
	AnswerCount int    `json:"answerCount"`
	Start       int    `json:"start"`
	Count       int    `json:"count"`
	SortType    string `json:"sortType"`
	Lang        string `json:"lang"`
	Answers     []struct {
		Qid              string      `json:"qid"`
		ID               string      `json:"id"`
		Text             string      `json:"text"`
		AttachedImageURL interface{} `json:"attachedImageUrl"` // todo find the right types
		AttachedImageID  interface{} `json:"attachedImageId"`  // todo find the right types
		Reference        interface{} `json:"reference"`        // todo find the right types
		Answerer         struct {
			Euid     string `json:"euid"`
			Kid      string `json:"kid"`
			Nickname string `json:"nickname"`
			ImageURL string `json:"imageUrl"`
			Level    int    `json:"level"`
		} `json:"answerer"`
		IsBestAnswer       bool      `json:"isBestAnswer"`
		ThumbsDown         int       `json:"thumbsDown"`
		ThumbsUp           int       `json:"thumbsUp"`
		IsAnonymous        bool      `json:"isAnonymous"`
		CommentCount       int       `json:"commentCount"`
		CreatedTime        time.Time `json:"createdTime"`
		UserAnswerRelation struct {
			HasFlagged          bool `json:"hasFlagged"`
			CanFlag             bool `json:"canFlag"`
			CanVote             bool `json:"canVote"`
			CanChooseBestAnswer bool `json:"canChooseBestAnswer"`
			HasVoted            bool `json:"hasVoted"`
			IsAuthor            bool `json:"isAuthor"`
			HasCommented        bool `json:"hasCommented"`
			CanComment          bool `json:"canComment"`
		} `json:"userAnswerRelation"`
	} `json:"answers"`
	IsServerFetched bool `json:"isServerFetched"`
}

func newQuestionPost(qid string, question string, text string, url string) forum.Post {
	return forum.Post{
		Url:          url,
		ThreadId:     qid,
		PostSeq:      0,
		PageSeq:      0,
		ThreadPostId: "",
		Id:           qid + "-q",
		User:         "",
		UserIcon:     "",
		Hdr:          question,
		Msg:          text,
		Html:         "",
	}
}

func newAnswerPost(qid string, nr int, text string, url string) forum.Post {
	return forum.Post{
		Url:          url,
		ThreadId:     qid,
		PostSeq:      nr + 1,
		PageSeq:      nr / 10,
		ThreadPostId: fmt.Sprintf("%s-%05d", qid, nr),
		Id:           qid + "-a-" + fmt.Sprintf("%05d", nr),
		User:         "",
		UserIcon:     "",
		Hdr:          "",
		Msg:          text,
		Html:         "",
	}

}

func (fp *YahooAnswersParser) ParseResponse(body io.Reader, header http.Header, uri string) ([]forum.Post, error) {
	// sanitizer := bluemonday.UGCPolicy()

	// There are special PUT requests that are used for pagination, we handle them here
	if strings.Contains(uri, "_reservice_") {
		return fp.parseReservice(body, uri)
	}

	ctype := header.Get("content-type")
	//fmt.Println("RSPTYPE", uri, ctype)
	if !strings.HasPrefix(ctype, "text/html") {
		return nil, fmt.Errorf("%s not text/html, but %s", uri, ctype)
	}

	root, err := html.Parse(body)
	if err != nil {
		log.Println("error parsing response body", err)
		return nil, err
	}

	threadUrl, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url %s: %w", uri, err)
	}
	// we need a unique identifier for this thread, we use the QID for this
	threadIdStr := threadUrl.Query().Get("qid")
	if threadIdStr == "" {
		return nil, fmt.Errorf("failed to get qid for %s", uri)
	}

	// fmt.Println("got qid page for", threadIdStr)

	//pageSeqStr := threadUrl.Query().Get("page")
	//pageSeq, err := strconv.Atoi(pageSeqStr)
	//
	//if err != nil {
	//	// fmt.Println("failed to parse thread identifier for URL", uri)
	//	// fmt.Println(threadUrl.Query())
	//	pageSeq = 1
	//}
	////fmt.Println(threadUrl.Query())

	doc := goquery.NewDocumentFromNode(root) // not sure where to pass URI.. the internal constructor supports it, but it is not available to us

	bodies := make([]forum.Post, 0)

	// for answer pages with <10 answers, the answers are embedded as a json-ld payload in the served page
	doc.Find("script[type=\"application/ld+json\"]").Each(func(i int, selection *goquery.Selection) {
		var payload PagePayload
		err := json.Unmarshal([]byte(selection.Text()), &payload)
		if err != nil {
			fmt.Println("failed to parse page payload", err)
			return
		}
		if payload.Type == "QAPage" {
			// fmt.Println("!!! GOT A SCRIPT HURRAH!", payload)
			// fmt.Println("QQQ", selection.Text())
			// put the accepted answer at position 0?
			question := payload.Mainentity.Name
			questionText := payload.Mainentity.Text
			// fmt.Println("QST", threadIdStr, question)
			bodies = append(bodies, newQuestionPost(threadIdStr, html.UnescapeString(question), html.UnescapeString(questionText), uri))
			bodies = append(bodies, newAnswerPost(threadIdStr, 0, html.UnescapeString(payload.Mainentity.Acceptedanswer.Text), uri))
			for nr, answer := range payload.Mainentity.Suggestedanswer {
				// TODO: get the start and count from the original request, so we know what the proper subpageSeq numbers are
				// fmt.Println("ANS2", threadIdStr, nr+1, html.UnescapeString(answer.Text))
				bodies = append(bodies, newAnswerPost(threadIdStr, nr+1, html.UnescapeString(answer.Text), uri))
			}

		}
	})

	//doc.Find(".content-border").Each(func(postNr int, s *goquery.Selection) {
	//	// For each item found, get the band and title
	//	id, _ := s.Attr("id")
	//	user := s.Find(".post-user").Text()
	//	msg := s.Find(".post-message").Text()
	//	hdr := s.Find(".post-header").Text()
	//	userIconUri, _ := s.Find(".post-avatar").Find(".avatar").Find("img").Attr("src")
	//	// fmt.Println(userIconUri, ok)
	//
	//	_ = sanitizer
	//	ohtml, _ := goquery.OuterHtml(s.Find(".post-message")) // todo handle error
	//	// todo apply transformation rules to modify html (or store the unsanitized html instead, and sanitize on retrieval
	//	sanehtml := sanitizer.Sanitize(ohtml)
	//	if len(msg) > 0 {
	//		// fmt.Printf("Post %s [%d]: %s : %s - %s\n", id, postNr, user, len(hdr), len(msg))
	//		x := forum.Post{
	//			Url:          uri,
	//			ThreadId:     threadIdStr,
	//			PageSeq:      pageSeq,
	//			PostSeq:      postNr,
	//			ThreadPostId: int64(pageSeq)*1000 + int64(postNr),
	//			Id:           id,
	//			User:         user,
	//			UserIcon:     userIconUri,
	//			Hdr:          hdr,
	//			Msg:          msg,
	//			Html:         sanehtml,
	//		}
	//		bodies = append(bodies, x)
	//	}
	//})
	return bodies, nil
}

func (fp *YahooAnswersParser) parseReservice(body io.Reader, url string) ([]forum.Post, error) {
	bodies := make([]forum.Post, 0)

	// fmt.Println("RESERVICE PARSE")
	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err // todo: wrap error
	}
	// fmt.Println(string(data))
	// var payload map[string]interface{}
	var payload ServicePayload
	err = json.Unmarshal(data, &payload)
	if err != nil {
		return nil, fmt.Errorf("reservice stage 1 unmarshall error: %s", err)
	}
	if payload.Type == "FETCH_QUESTION_ANSWERS_END" {
		var qapayload QAPagePayload
		err = json.Unmarshal(payload.Payload, &qapayload)
		if err != nil {
			return nil, fmt.Errorf("reservice stage 2 unmarshall error: %s", err)
		}
		qid := qapayload.Qid
		for answerNr, answer := range qapayload.Answers {
			pageSeq := qapayload.Start + answerNr
			// fmt.Println("ANS", qid, pageSeq, answer.Text)

			// TODO refactor and emit Posts to output
			bodies = append(bodies, newAnswerPost(qid, pageSeq, answer.Text, url))

		}
	}
	return nil, nil
}

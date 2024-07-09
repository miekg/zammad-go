package main

import (
	"bytes"
	"fmt"

	"github.com/AlessandroSechi/zammad-go"
	"github.com/anacrolix/fuse"
)

func (f *File) ArticleContent() ([]byte, error) {
	buf := &bytes.Buffer{}
	a, err := f.z.TicketArticleByTicket(f.Ticket.ID)
	if err != nil {
		return nil, err
	}
	*f.articles = a

	for i, a := range *f.articles {
		if i > 0 {
			fmt.Fprintln(buf, "\n*********************************")
		}
		fmt.Fprintf(buf, "From: %s\n", a.From)
		fmt.Fprintf(buf, "To: %s\n", a.To)
		if a.Subject != nil {
			fmt.Fprintf(buf, "Subject: %s\n", a.Subject)
		}
		if a.Internal {
			fmt.Fprintf(buf, "internal\n")
		} else {
			fmt.Fprintf(buf, "public\n")
		}
		fmt.Fprintf(buf, "\n%s\n", a.Body)
	}
	return buf.Bytes(), nil
}

func (f *File) TagContent() ([]byte, error) {
	ta, err := f.z.TicketTagByTicket(f.Ticket.ID)
	if err != nil {
		return nil, err
	}
	*f.tags = ta
	buf := &bytes.Buffer{}
	for _, s := range *f.tags {
		fmt.Fprintf(buf, "%s\n", s.Name)
	}
	return buf.Bytes(), nil
}

func ArticleWrite(t zammad.Ticket, req *fuse.WriteRequest) zammad.TicketArticle {
	id := req.Uid
	// Use FromFunc here.

	ta := zammad.TicketArticle{}
	ta.TicketID = t.ID
	ta.TypeID = 10 // miek: this a note.. via https://docs.zammad.org/en/latest/api/ticket/articles.html#general-information-about-ticket-articles
	ta.OriginByID = id
	ta.Subject = "From ticketfs"
	ta.SenderID = 2 // agent?
	ta.UpdatedByID = int(id)
	ta.CreatedByID = int(id)
	ta.Body = string(req.Data)
	ta.Internal = true
	ta.From = "ticketfs"
	ta.ContentType = "text/plain"

	return ta
}

package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/AlessandroSechi/zammad-go"
	"go.science.ru.nl/log"
)

// TicketState contains the ticket states and is filled on startup
var TicketState map[int]string

func state(token, url string) {
	TicketState = map[int]string{}
	z := NewZammad(token, url)
	states, err := z.TicketStateList()
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range states {
		TicketState[s.ID] = s.Name
	}
}

func NewZammad(token, url string) *zammad.Client {
	client := &zammad.Client{
		Token:  token,
		Url:    url,
		Client: &http.Client{Timeout: 5 * time.Second},
	}
	return client
}

func ParseUint(s string) uint64 {
	j, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return j
}

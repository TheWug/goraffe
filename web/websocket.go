package web

import (
	"github.com/thewug/goraffe/store"
)

type RaffleHub struct {
	Raffle  store.Raffle

	Clients map[int][]*Client
	Masters []*Client

	Register chan *Client
	Unregister chan *Client

	Actions chan *Action
}

type Action struct {
	J []byte
	Client *Client
}

type Status struct {
	Type string `json:"type"`
}

type Lose struct {
	Type string `json:"type"`
	Winner string `json:"winner"`
}

type MasterStatus struct {
	Type string `json:"type"`
	Id   int    `json:"id"`
	Name string `json:"name,omitempty"`
}

type Client struct {
}

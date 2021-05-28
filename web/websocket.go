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
}

type Status struct {
}

type Lose struct {
}

type MasterStatus struct {
}

type Client struct {
}

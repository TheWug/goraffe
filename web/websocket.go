package web

import (
	"github.com/gorilla/websocket"

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

var all_hubs map[string]*RaffleHub = make(map[string]*RaffleHub)

func LookupRaffleHub(raffle_id string) *RaffleHub {
	if hub, ok := all_hubs[raffle_id]; ok && hub != nil {
		return hub
	}

	var h RaffleHub
	err := store.Transact(&h.Raffle, raffle_id, store.GetRaffle)
	if err != nil {
		return nil
	}

	h = RaffleHub{
		Raffle: h.Raffle,
		Clients: make(map[int][]*Client),
		Register: make(chan *Client),
		Unregister: make(chan *Client),
		Actions: make(chan *Action),
	}

	go h.Run()

	all_hubs[raffle_id] = &h
	return &h
}

type Client struct {
	Hub     *RaffleHub

	Id       int
	Name     string

	Conn    *websocket.Conn

	Outgoing chan []byte
}

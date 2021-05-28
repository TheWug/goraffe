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

func (this *RaffleHub) Run() {
}

func (this *RaffleHub) SendTo(client *Client, data []byte) {
}

func (this *RaffleHub) TargetedBroadcast(to_id int, status, master_status interface{}) {
}

func (this *RaffleHub) Broadcast(mode string) {
}

func (this *RaffleHub) TargetedGeneric(client *Client, fn func(int, string) (bool, error), mode string) {
}

func (this *RaffleHub) Enter(client *Client) {
}

func (this *RaffleHub) Withdraw(client *Client) {
}

func (this *RaffleHub) Disqualify(client *Client, to_dq int) {
}

func (this *RaffleHub) Undisqualify(client *Client, to_undq int) {
}

func (this *RaffleHub) Generic(fn func() (bool, error), mode string) {
}

func (this *RaffleHub) Open() {
}

func (this *RaffleHub) Close() {
}

func (this *RaffleHub) Cancel() {
}

func (this *RaffleHub) Draw() {
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

const (
	maxMessageSize = 340
	writeWait = 10 * time.Second
	pongWait = 300 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

func (this *Client) readPump() {
}

func (this *Client) writePump() {
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

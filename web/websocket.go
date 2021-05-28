package web

import (
	"net/http"
	"time"

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
	for {
	select {
	case c := <- this.Register:
		c.Hub = this
		this.Clients[c.Id] = append(this.Clients[c.Id], c)
		if c.Id == this.Raffle.Owner {
			this.Masters = append(this.Masters, c)
		}

		go c.readPump()
		go c.writePump()
		entry, err := this.Raffle.Status(c.Id)
		if err != nil {
			// XXX log
		}
		if entry == nil {
		} else if entry.Disqualified {
			b, _ := json.Marshal(Status{Type: "disqualify"})
			c.Outgoing <- b
		} else if entry.Entered {
			b, _ := json.Marshal(Status{Type: "enter"})
			c.Outgoing <- b
		} else {
			b, _ := json.Marshal(Status{Type: "withdraw"})
			c.Outgoing <- b
		}

		if this.Raffle.IsOpen {
			b, _ := json.Marshal(Status{Type: "open"})
			c.Outgoing <- b
		} else {
			b, _ := json.Marshal(Status{Type: "close"})
			c.Outgoing <- b
		}
	case c := <- this.Unregister:
		list, ok := this.Clients[c.Id]
		if ok {
			for i, v := range list {
				if c == v {
					list[i] = list[len(list) - 1]
					if len(list) > 1 {
						this.Clients[c.Id] = list[:len(list) - 1]
					} else {
						delete(this.Clients, c.Id)
					}
					break
				}
			}
		}
		for i, v := range this.Masters {
			if c == v {
				this.Masters[i] = this.Masters[len(list) - 1]
				this.Masters = this.Masters[:len(list) - 1]
				break
			}
		}
		close(c.Outgoing)
		c.Hub = nil
	case a := <- this.Actions:
		var s Status
		err := json.Unmarshal(a.J, &s)
		if err != nil {
			break // XXX log
		}

		type DQ struct {
			Type string
			TargetId int
		}

		switch s.Type {
		case "enter":
			if this.Raffle.IsOpen {
				this.Enter(a.Client)
			}
		case "withdraw":
			if this.Raffle.IsOpen {
				this.Withdraw(a.Client)
			}
		default: // all other cases require raffle ownership
			if a.Client.Id != this.Raffle.Owner {
				break
			}
			switch s.Type {
			case "disqualify":
				var dq DQ
				err := json.Unmarshal(a.J, &dq)
				if err != nil {
					break // XXX log
				}
				this.Disqualify(a.Client, dq.TargetId)
			case "undisqualify":
				var dq DQ
				err := json.Unmarshal(a.J, &dq)
				if err != nil {
					break // XXX log
				}
				this.Undisqualify(a.Client, dq.TargetId)
			case "open":
				this.Open()
			case "close":
				this.Close()
			case "cancel":
				this.Cancel()
			case "draw":
				this.Draw()
			}
		}
	} // select
	} // for
}

func (this *RaffleHub) SendTo(client *Client, data []byte) {
	client.Outgoing <- data
}

func (this *RaffleHub) TargetedBroadcast(to_id int, status, master_status interface{}) {
	st, _ := json.Marshal(status)
	mst, _ := json.Marshal(master_status)

	for _, c := range this.Clients[to_id] {
		this.SendTo(c, st)
	}
	for _, c := range this.Masters {
		this.SendTo(c, mst)
	}
}

func (this *RaffleHub) Broadcast(mode string) {
	st, _ := json.Marshal(Status{Type: mode})

	for _, v := range this.Clients {
		for _, c := range v {
			this.SendTo(c, st)
		}
	}
}

func (this *RaffleHub) TargetedGeneric(client *Client, fn func(int, string) (bool, error), mode string) {
	changed, err := fn(client.Id, client.Name)
	if err != nil {
		// XXX log error
		return
	} else if !changed {
		return
	} else {
		this.TargetedBroadcast(client.Id, Status{Type: mode}, MasterStatus{Type: "notify-" + mode, Id: client.Id, Name: client.Name})
	}
}

func (this *RaffleHub) Enter(client *Client) {
	this.TargetedGeneric(client, this.Raffle.Enter, "enter")
}

func (this *RaffleHub) Withdraw(client *Client) {
	this.TargetedGeneric(client, this.Raffle.Withdraw, "withdraw")
}

func (this *RaffleHub) Disqualify(client *Client, to_dq int) {
	changed, err := this.Raffle.Disqualify(to_dq)
	if err != nil {
		// XXX log error
		return
	} else if !changed {
		return
	} else {
		this.TargetedBroadcast(client.Id, Status{Type: "disqualify"}, MasterStatus{Type: "notify-disqualify", Id: to_dq})
	}
}

func (this *RaffleHub) Undisqualify(client *Client, to_undq int) {
	entry, err := this.Raffle.Undisqualify(to_undq)
	if err != nil {
		// XXX log error
		return
	} else if entry.Entered {
		this.TargetedBroadcast(client.Id, Status{Type: "enter"}, MasterStatus{Type: "notify-enter", Id: to_undq, Name: entry.Name})
	} else {
		this.TargetedBroadcast(client.Id, Status{Type: "withdraw"}, MasterStatus{Type: "notify-withdraw", Id: to_undq, Name: entry.Name})
	}
}

func (this *RaffleHub) Generic(fn func() (bool, error), mode string) {
	changed, err := fn()
	if err != nil {
		// XXX log error
		return
	} else if !changed {
		return
	} else {
		this.Broadcast(mode)
	}
}

func (this *RaffleHub) Open() {
	this.Generic(this.Raffle.Open, "open")
}

func (this *RaffleHub) Close() {
	this.Generic(this.Raffle.Close, "close")
}

func (this *RaffleHub) Cancel() {
	this.Generic(this.Raffle.Cancel, "reset")
}

func (this *RaffleHub) Draw() {
	winner, err := this.Raffle.Draw()
	if err != nil {
		// XXX log error
		return
	} else if winner == nil {
		return
	}

	win, _ := json.Marshal(Status{Type: "win"})
	lose, _ := json.Marshal(Lose{Type: "lose", Winner: winner.Name})

	for k, v := range this.Clients {
		if k == winner.UserId {
			for _, c := range v {
				this.SendTo(c, win)
			}
		} else {
			for _, c := range v {
				this.SendTo(c, lose)
			}
		}
	}
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
	defer func() {
		this.Hub.Unregister <- this
		this.Conn.Close()
	}()
	this.Conn.SetReadLimit(maxMessageSize)
	this.Conn.SetReadDeadline(time.Now().Add(pongWait))
	this.Conn.SetPongHandler(func(string) error { this.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := this.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			}
			break
		}
		this.Hub.Actions <- &Action{J: message, Client: this}
	}
}

func (this *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		this.Conn.Close()
	}()
	for {
		select {
		case message, ok := <- this.Outgoing:
			this.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				this.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := this.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			this.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := this.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func WebSocket(w http.ResponseWriter, req *http.Request) {
	login := auth.Get(req)
	if login == nil {
		return
	}

	raffle_id := strings.TrimPrefix(req.URL.Path, fmt.Sprintf(PATH_WEBSOCKET, ""))
	if raffle_id == "" {
		return
	}

	user, err := patreon.GetUserInfo(&login.Patreon)
	if err == patreon.BadLogin {
		auth.Delete(w)
		RedirectLinkAccountAndReturn(w, req)
		return
	} else if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	hub := LookupRaffleHub(raffle_id)
	if hub == nil {
		return
	}

	conn, err := upgrader.Upgrade(w, req, nil)

	client := Client{
		Id: user.Id,
		Name: user.FullName,
		Conn: conn,
		Outgoing: make(chan []byte, 32),
	}

	hub.Register <- &client
}

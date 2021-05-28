package main

import (
	"fmt"
	"strconv"
	"log"
	"io"
	"net/http"
	"net/url"
	"time"
	"html/template"
	"encoding/json"
	"io/ioutil"
	"strings"
	"sort"
	"database/sql"

	"github.com/thewug/goraffe/web"
	"github.com/thewug/goraffe/auth"
	"github.com/thewug/goraffe/patreon"
	"github.com/thewug/goraffe/store"
)

type ClientSettings struct {
	PatreonApiClientId     string `json:"patreon_api_client_id"`
	PatreonApiClientSecret string `json:"patreon_api_client_secret"`
	PatreonLoginRedirect   string `json:"patreon_login_redirect"`
	DatabaseUrl            string `json:"database_url"`
}

func GetClientSettings() ClientSettings {
	var settings ClientSettings
	bytes, err := ioutil.ReadFile("./settings.json")
	if err != nil {
		log.Fatal("Read settings file: ", err.Error())
	}

	json.Unmarshal(bytes, &settings)
	return settings
}

func templateWrite(w io.WriteCloser, t *template.Template, data interface{}) {
	t.Execute(w, data)
	w.Close()
}

func LinkAccount(w http.ResponseWriter, req *http.Request) {
	state_plain := auth.PatreonState{
		ReturnTo: req.URL.Query().Get("returnto"),
		IV: "9ryhar9sreaskt60j3m54",
	}

	state, err := auth.EncryptAndSign(state_plain)
	if err != nil {
		http.Error(w, fmt.Sprintf("Encryption: %s", err.Error()), 500)
		return
	}

	templ := template.Must(template.New("connecttopatreonpage").Parse(
`<html></head><title>Connect to Patreon</title></head><body>
<form method="get" action="https://www.patreon.com/oauth2/authorize">
<input type="hidden" name="response_type" value="code">
<input type="hidden" name="client_id" value="{{.Settings.PatreonApiClientId}}">
<input type="hidden" name="redirect_uri" value="{{.Settings.PatreonLoginRedirect}}">
<input type="hidden" name="scope" value="identity identity.memberships campaigns campaigns.members">
<input type="hidden" name="state" value="{{.State}}">
<input type="submit" value="Connect to your Patreon account">
</form></body></html>`,
))

	rp, wp := io.Pipe()
	go templateWrite(wp, templ, map[string]interface{} {
		"State": state,
		"Settings": GetClientSettings(),
	})

	io.Copy(w, rp)
	rp.Close()
}

func AboutPage(w http.ResponseWriter, req *http.Request) {
	templ := template.Must(template.New("aboutpage").Parse(
`<html></head><title>About Goraffe</title></head><body>
<h1>About Goraffe</h1>
<p>Patreon doesn't allow raffles, because it is considered a form of gambling. But that makes it difficult to distribute a limited quantity reward among a tier with more people than it can easily be divided between.</p>
<p>And that's where Goraffe comes in: Goraffe (pronounced like "giraffe") is a raffle-like system, designed to be compatible with Patreon's rules surrounding raffles. The key difference, and the reason Goraffe is allowed on Patreon while true raffles are not, is that with a true raffle, the winning draw is completely random.  Goraffe uses a deterministic probability model with the same long-term outcomes as a true raffle, but in the short term, wins are divided fairly amongst everyone who participates.</p>
<p>It has the same look-and-feel as a true raffle, and there is still a bit of luck to it, so hopefully it's still fun.</p>
</body></html>`,
))

	rp, wp := io.Pipe()
	go templateWrite(wp, templ, nil)

	io.Copy(w, rp)
	rp.Close()
}

func RaffleDashboard(w http.ResponseWriter, req *http.Request) {
	templ := template.Must(template.New("dashboardpage").Parse(
`<html></head><title>Dashboard</title></head><body>
{{if ne (len .MyRaffles) 0}}<h1>Your Raffles</h1>{{end}}
{{range .MyRaffles}}<a href="/r/{{.Id}}">{{.Display}}</a><br/>{{end}}
<form action="/new" method="get"><input type="submit" value="Create a new raffle"></form><br/>
{{if ne (len .EnteredRaffles) 0}}<h1>Raffles You've Entered Before</h1>{{end}}
{{range .EnteredRaffles}}<a href="/r/{{.Id}}">{{.Display}}</a><br/>{{end}}
</body></html>`,
	))

	login := auth.Get(req)
	if login == nil {
		web.RedirectLinkAccountAndReturn(w, req)
		return
	}

	user, err := patreon.GetUserInfo(&login.Patreon)
	if err == patreon.BadLogin {
		auth.Delete(w)
		web.RedirectLinkAccountAndReturn(w, req)
		return
	} else if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var my_raffles, entered_raffles []store.Raffle
	err = store.Transact(nil, nil, func(tx *sql.Tx, x, y interface{}) (error) {
		err := store.GetMyRaffles(tx, &my_raffles, user.Id)
		if err != nil {
			return err
		}
		err = store.GetEnteredRaffles(tx, &entered_raffles, user.Id)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	args := map[string]interface{}{
		"MyRaffles": my_raffles,
		"EnteredRaffles": entered_raffles,
	}

	rp, wp := io.Pipe()
	go templateWrite(wp, templ, args)

	io.Copy(w, rp)
	rp.Close()
}

func LinkAccountPatreonReturn(w http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	var state auth.PatreonState
	err := auth.DecryptAndValidate(q.Get("state"), &state)
	if err != nil {
		// XXX logging should occur here, as it may indicate attempted abuse
		// intentionally use the same, misleading error message as for expired state
		http.Error(w, "Expired state", 400)
		return
	}

	// XXX also check if state is expired

	v := url.Values{}
	settings := GetClientSettings()
	v.Set("code", q.Get("code"))
	v.Set("grant_type", "authorization_code")
	v.Set("client_id", settings.PatreonApiClientId)
	v.Set("client_secret", settings.PatreonApiClientSecret)
	v.Set("redirect_uri", settings.PatreonLoginRedirect)
	resp, err := http.PostForm("https://www.patreon.com/api/oauth2/token", v)
	if err != nil {
		http.Error(w, "Error connecting to patreon 1", 502)
		return
	}

	defer resp.Body.Close()

	var s auth.Session
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Bad response from patreon 2", 502)
		return
	}

	err = json.Unmarshal(b, &s.Patreon)
	if err != nil {
		http.Error(w, "Bad response from patreon 3", 502)
		return
	}

	s.SessionDate = time.Now()

	auth.Put(w, &s)
	templ := template.Must(template.New("loggedinpage").Parse(
`<html><head><title>Account Connected</title>
{{if .HasState}}<meta http-equiv="refresh" content="3;url={{.ReturnTo}}" />{{end}}
</head><body><p>Your patreon account is now connected.</p></body></html>`,
	))

	rp, wp := io.Pipe()
	go templateWrite(wp, templ, map[string]interface{} {
		"HasState": state.ReturnTo != "",
		"ReturnTo": state.ReturnTo,
	})

	io.Copy(w, rp)
	rp.Close()
}

func NewRaffle(w http.ResponseWriter, req *http.Request) {
	m := strings.ToUpper(req.Method)
	if m == "GET" {
		NewRaffleGet(w, req)
	} else if m == "POST" {
		NewRafflePost(w, req)
	} else {
		http.Error(w, "Invalid method", 400)
		return
	}
}

func NewRaffleGet(w http.ResponseWriter, req *http.Request) {
	templ := template.Must(template.New("newrafflepage").Parse(
`<html><head>
<title>New Raffle</title>
<script src="https://ajax.googleapis.com/ajax/libs/jquery/3.5.1/jquery.min.js"></script>
<script src="/raffle.js"></script>
</head><body>
<p>Welcome {{.Title}}!<br>Create a new raffle here.</p>
<form action="#" method="post">
<label>Which patron tiers should be allowed to enter the raffle?</label><br />
{{range .Tiers}}<input onclick="selectTiers({{.ContributionCents}})" class="tier" type="checkbox" id="{{.ContributionCents}}" name="{{.ContributionCents}}" {{if eq .ContributionCents 0}}un{{end}}checked>
<label for="{{.ContributionCents}}">{{.Name}}{{if ne .ContributionCents 0}} ({{call $.Format .ContributionCents}}){{end}}</label>
<br />{{end}}
<label for="raffle_name">Raffle Name:</label><input type="text" id="raffle_name" name="raffle_name"><br/>
<input type="submit" value="Create Raffle">
</body></html>`,
	))

	login := auth.Get(req)
	if login == nil {
		web.RedirectLinkAccountAndReturn(w, req)
		return
	}

	title, tiers, err := patreon.GetTitleAndTiers(&login.Patreon)
	if err == patreon.BadLogin {
		auth.Delete(w)
		web.RedirectLinkAccountAndReturn(w, req)
		return
	} else if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	_ = tiers

	rp, wp := io.Pipe()
	go templateWrite(wp, templ, nil)

	auth.Put(w, login)
	io.Copy(w, rp) // XXX listen for errors
	rp.Close()
}

func NewRafflePost(w http.ResponseWriter, req *http.Request) {
	login := auth.Get(req)
	if login == nil {
		web.RedirectLinkAccountAndReturn(w, req)
		return
	}

	var tiers []int32
	req.ParseForm()
	for k, _ := range req.PostForm {
		if i, e := strconv.Atoi(k); e == nil {
			tiers = append(tiers, int32(i))
		}
	}

	name := req.PostFormValue("raffle_name")

	user, err := patreon.GetUserInfo(&login.Patreon)
	if err == patreon.BadLogin {
		auth.Delete(w)
		web.RedirectLinkAccountAndReturn(w, req)
		return
	} else if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	raffle, err := store.CreateRaffle(user.Id, name, tiers)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	web.RedirectToRaffle(w, req, raffle)
}

func main() {
	fmt.Println("goraffe!")
	http.HandleFunc(web.PATH_NEW_RAFFLE, NewRaffle)
	http.HandleFunc(web.PATH_LINK_ACCOUNT, LinkAccount)
	http.HandleFunc(web.PATH_ACCOUNT_LINKING, LinkAccountPatreonReturn)
	err := http.ListenAndServe(":3001", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}

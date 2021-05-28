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
		fmt.Println(err.Error())
		http.Error(w, "Bad response from patreon 3", 502)
		return
	}

	log.Println(string(b))
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
	templ := template.Must(template.New("newrafflepage").Parse(
`<html><head><title>New Raffle</title></head><body>
<p>Create a new raffle here.</p>
<form action="#">
<input>
</body></html>`,
))

	login := auth.Get(req)
	if login == nil {
		web.RedirectLinkAccountAndReturn(w, req)
		return
	}

	tiers, err := patreon.GetCampaignTiers(&login.Patreon)
	if err == patreon.BadLogin {
		auth.Delete(w)
		web.RedirectLinkAccountAndReturn(w, req)
		return
	}
	
	_ = tiers

	rp, wp := io.Pipe()
	go templateWrite(wp, templ, nil)

	auth.Put(w, login)
	io.Copy(w, rp) // XXX listen for errors
	rp.Close()
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

package main

import (
	"fmt"
	"log"
	"io"
	"net/http"
	"net/url"
	"time"
	"html/template"
	"encoding/json"
	"io/ioutil"

	"github.com/thewug/goraffe/web"
	"github.com/thewug/goraffe/auth"
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
	templ := template.Must(template.New("connecttopatreonpage").Parse(
`<html></head><title>Connect to Patreon</title></head><body>
<form method="get" action="https://www.patreon.com/oauth2/authorize">
<input type="hidden" name="response_type" value="code">
<input type="hidden" name="client_id" value="{{.PatreonApiClientId}}">
<input type="hidden" name="redirect_uri" value="{{.PatreonLoginRedirect}}">
<input type="hidden" name="scope" value="identity identity.memberships campaigns campaigns.members">
<input type="submit" value="Connect to your Patreon account">
</form></body></html>`,
))

	rp, wp := io.Pipe()
	go templateWrite(wp, templ)

	io.Copy(w, rp)
	rp.Close()
}

func NewRaffle(w http.ResponseWriter, req *http.Request) {
	templ := template.Must(template.New("newrafflepage").Parse(
`<html><head><title>New Raffle</title></head><body><p>Create a new raffle here!</body></html>`,
))

	login := auth.Get(req)
	if login == nil {
		web.RedirectLinkAccountAndReturn(w, req)
		return
	}

	rp, wp := io.Pipe()
	go templateWrite(wp, templ)

	auth.Put(w, login)
	io.Copy(w, rp) // XXX listen for errors
	rp.Close()
}

func main() {
	fmt.Println("goraffe!")
	http.HandleFunc(web.PATH_NEW_RAFFLE, NewRaffle)
	http.HandleFunc(web.PATH_LINK_ACCOUNT, LinkAccount)
	err := http.ListenAndServe(":3001", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}

package main

import (
	"fmt"
	"log"
	"io"
	"net/http"
	"html/template"

	"github.com/thewug/goraffe/web"
	"github.com/thewug/goraffe/auth"
)

type ClientSettings struct {
	PatreonApiClientId string
	PatreonLoginRedirect string
}

func GetClientSettings() ClientSettings {
	return ClientSettings{
		PatreonApiClientId: "wv473kvTpLjcUliP7aj7JAOYxKgCWefEagZpNsercCE_EmSvVJcRJv_-B_PCIeX8",
		PatreonLoginRedirect: "https://local.wuggl.es/login_redirect",
	}
}

func templateWrite(w io.WriteCloser, t *template.Template) {
	t.Execute(w, GetClientSettings())
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

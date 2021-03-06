package web

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/thewug/goraffe/store"
)

const (
	PATH_LINK_ACCOUNT    = "/patreon_login"
	PATH_ACCOUNT_LINKING = "/patreon_return"
	PATH_NEW_RAFFLE      = "/new"
	PATH_RAFFLE          = "/r/%s"
	PATH_WEBSOCKET       = "/ws/%s"
	PATH_DASHBOARD       = "/dashboard"
	PATH_ABOUT           = "/about"
	PATH_SCRIPT          = "/raffle.js"
)

func RedirectLinkAccountAndReturn(w http.ResponseWriter, req *http.Request) {
	return_to := req.URL.Path
	if !strings.HasPrefix(return_to, "/") {
		http.Error(w, "Refusing to obey relative redirect", 400)
		return
	}
	new_url := fmt.Sprintf("%s?returnto=%s", PATH_LINK_ACCOUNT, html.EscapeString(return_to))
	http.Redirect(w, req, new_url, 303)
}

func RedirectToRaffle(w http.ResponseWriter, req *http.Request, raffle *store.Raffle) {
	new_url := fmt.Sprintf(PATH_RAFFLE, html.EscapeString(raffle.Id))
	http.Redirect(w, req, new_url, 303)
}

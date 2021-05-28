package web

import (
	"net/http"
	"html"
	"strings"
	"fmt"
)

const (
	PATH_LINK_ACCOUNT    = "/patreon_login"
	PATH_ACCOUNT_LINKING = "/patreon_return"
	PATH_NEW_RAFFLE      = "/new"
	PATH_RAFFLE          = "/r/"
	PATH_DASHBOARD       = "/dashboard"
	PATH_ABOUT           = "/about"
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

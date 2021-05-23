# goraffe
A patreon raffle machine written on gorilla.

Structure:

Web Server / Reverse Proxy <-------- clients
        |                            /
        |                           /
        v                          v
    goraffe ---------------> patreon api
        
clients interact primarily with the frontend web server. This server serves static assets which can be modified or customized at will.
clients will interact with patreon's api for authentication purposes.
frontend web server forwards web socket communication to goraffe.
goraffe handles client raffle status changes and requests for information.
goraffe looks up client account status info using patreon's api.

URL structure:
domain.com/new - create a new raffle
domain.com/patreonlogin - patreon account connection
domain.com/r/XXXXXXXXXX - raffle page

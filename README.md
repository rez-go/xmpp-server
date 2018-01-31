# Experimental XMPP Server

This project was a tool for me to dive into XMPP, again.

It was implemented from scratch with inspiration from [Russ Cox's
`go-xmpp` library](https://github.com/mattn/go-xmpp/).

The goal was to implement the functionality so it's possible to do
one-to-one chat. And yes, if you are new to XMPP, you can see from what's
already implemented that XMPP is fundamentally complex and verbose.

Currently, the server has no data store backend at all. For user
authentication, it depends on an OAuth 2.0 server. The authentication is
performed using the resource owner password credentials grant flow.

TLS is currently not supported.

## Giving it a Try

If you really want to try this, first, you'll need to know the basics
of building a Go project. Then locally clone the project, and you'll
need to edit `cmd/xmpp-server/server.go`. Modify `OAuthTokenEndpoint`,
`OAuthClientID` and `OAuthClientSecret` with values obtained from the
OAuth server (the server must be able to perform authentication using
resource owner password credentials grant flow). Build and run:
`go build ./cmd/xmpp-server && ./xmpp-server`.

Tested with these clients:

- Adium
- Swift
- OneTeam
- Profanity

## The Code

The code is still full of panics as the placeholder for the proper
error handling.

It won't pass `golint`.

Test coverage is definitely very low.

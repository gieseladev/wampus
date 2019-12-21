package wampus

import "github.com/gammazero/nexus/v3/wamp"

const DiscordURIPrefix = "com.discord."

var (
	ErrURI           = wamp.URI(DiscordURIPrefix + "error")
	InternalErrorURI = ErrURI + ".internal"
	UnauthorizedURI  = ErrURI + ".unauthorized"
	NotFoundURI      = ErrURI + ".not_found"
)

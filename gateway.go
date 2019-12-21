package wampus

import (
	"context"
	"github.com/gammazero/nexus/v3/client"
	"github.com/gammazero/nexus/v3/wamp"
)

func (c *Component) registerGatewayProcedures() error {
	return joinErrors(
		c.wampClient.Register(DiscordURIPrefix+"update_voice_state", c.updateVoiceState, nil),
	)
}

func (c *Component) updateVoiceState(ctx context.Context, invocation *wamp.Invocation) client.InvokeResult {
	args := invocation.Arguments

	if len(args) == 0 {
		return resultFromErrorURI(wamp.ErrInvalidArgument, "guild id missing")
	}

	gID, _ := asSnowflake(args[0])
	if gID == "" {
		return resultFromErrorURI(wamp.ErrInvalidArgument, "guild id needs to be a snowflake")
	}

	var cID string
	if len(args) > 1 {
		cID, _ = asSnowflake(args[1])
	}

	kwargs := invocation.ArgumentsKw

	mute, _ := wamp.AsBool(kwargs["mute"])
	deaf, _ := wamp.AsBool(kwargs["deaf"])

	err := fnRaceContext(ctx, func() error {
		return c.discordSess.ChannelVoiceJoinManual(gID, cID, mute, deaf)
	})
	if err != nil {
		return resultFromErrorURI(InternalErrorURI, err.Error())
	}

	return emptyResult
}

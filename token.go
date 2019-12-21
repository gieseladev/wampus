package wampus

import (
	"context"
	"github.com/bwmarrin/discordgo"
	"github.com/gammazero/nexus/v3/client"
	"github.com/gammazero/nexus/v3/wamp"
)

func (c *Component) registerTokenProcedures() error {
	return joinErrors(
		c.wampClient.Register(DiscordURIPrefix+"token.user", tokenGetUser, nil),
		c.wampClient.Register(DiscordURIPrefix+"token.guilds", tokenGetGuilds, nil),
		c.wampClient.Register(DiscordURIPrefix+"token.in_guild", tokenIsInGuild, nil),
	)
}

func sessionForToken(args wamp.List) (*discordgo.Session, client.InvokeResult) {
	if len(args) < 1 {
		return nil, resultFromErrorURI(wamp.ErrInvalidArgument, "missing token")
	}

	token, ok := wamp.AsString(args[0])
	if !ok {
		return nil, resultFromErrorURI(wamp.ErrInvalidArgument, "arg 0 (token) must be a string")
	}
	sess, err := discordgo.New(token)
	if err != nil {
		// TODO this shouldn't (can't) happen so report an internal error!
		return nil, resultFromErrorURI(InternalErrorURI, err.Error())
	}

	return sess, client.InvokeResult{}
}

func tokenGetUser(ctx context.Context, invocation *wamp.Invocation) client.InvokeResult {
	sess, res := sessionForToken(invocation.Arguments)
	if sess == nil {
		return res
	}
	defer func() { _ = sess.Close() }()

	var u *discordgo.User
	err := fnRaceContext(ctx, func() error {
		var err error
		u, err = sess.User("@me")
		return err
	})
	if err != nil {
		return resultFromDiscordErr(err)
	}

	return client.InvokeResult{
		Args: wamp.List{u},
	}
}

func tokenGetGuilds(ctx context.Context, invocation *wamp.Invocation) client.InvokeResult {
	sess, res := sessionForToken(invocation.Arguments)
	if sess == nil {
		return res
	}
	defer func() { _ = sess.Close() }()

	var guilds []*discordgo.UserGuild
	err := fnRaceContext(ctx, func() error {
		var err error
		guilds, err = sess.UserGuilds(100, "", "")
		return err
	})
	if err != nil {
		return resultFromDiscordErr(err)
	}

	var args = make(wamp.List, len(guilds))
	for i, guild := range guilds {
		args[i] = guild
	}

	return client.InvokeResult{
		Args: args,
	}
}

func tokenIsInGuild(ctx context.Context, invocation *wamp.Invocation) client.InvokeResult {
	if len(invocation.Arguments) != 2 {
		return resultFromErrorURI(wamp.ErrInvalidArgument, "expected 2 arguments (token, guild id)")
	}

	guildID, ok := asSnowflake(invocation.Arguments[1])
	if !ok {
		return resultFromErrorURI(wamp.ErrInvalidArgument, "arg 1 (guild id) must be a snowflake")
	}

	sess, res := sessionForToken(invocation.Arguments)
	if sess == nil {
		return res
	}
	defer func() { _ = sess.Close() }()

	var guilds []*discordgo.UserGuild
	err := fnRaceContext(ctx, func() error {
		var err error
		guilds, err = sess.UserGuilds(100, "", "")
		return err
	})
	if err != nil {
		return resultFromDiscordErr(err)
	}

	found := false
	for _, guild := range guilds {
		if guild.ID == guildID {
			found = true
			break
		}
	}

	return client.InvokeResult{
		Args: wamp.List{found},
	}
}

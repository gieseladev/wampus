// package wampus provides a WAMP component for Discord.
package wampus

import (
	"context"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/gammazero/nexus/v3/client"
	"github.com/gammazero/nexus/v3/wamp"
	"strconv"
	"strings"
)

var (
	emptyResult = client.InvokeResult{}

	discordErrURI = wamp.URI("com.discord.error")
)

func resultFromErrorURI(uri wamp.URI, args ...interface{}) client.InvokeResult {
	return client.InvokeResult{
		Err:  uri,
		Args: args,
	}
}

func joinErrors(errs ...error) error {
	var lastErr error
	errStrings := make([]string, len(errs))

	for i, err := range errs {
		if err != nil {
			lastErr = err
			errStrings[i] = err.Error()
		}
	}

	if len(errStrings) == 1 {
		return lastErr
	} else if len(errStrings) > 1 {
		return errors.New(strings.Join(errStrings, "\n"))
	} else {
		return nil
	}
}

// asSnowflake is an extended type assertion similar to the ones provided
// by wamp, but it converts to discordgo Snowflakes (strings) which includes
// integers.
func asSnowflake(v interface{}) (string, bool) {
	if s, ok := wamp.AsString(v); ok {
		return s, true
	} else if i, ok := wamp.AsInt64(v); ok {
		return strconv.FormatInt(i, 10), true
	} else {
		return "", false
	}
}

// Component holds the discord session and the WAMP client.
type Component struct {
	discordSess *discordgo.Session
	wampClient  *client.Client
}

// NewComponent creates a new component from the given discord session and
// WAMP client.
func NewComponent(d *discordgo.Session, s *client.Client) *Component {
	return &Component{d, s}
}

// Connect creates a new component ...
func Connect(ctx context.Context, discordToken string, routerURL string, cfg client.Config) (*Component, error) {
	d, err := discordgo.New(fmt.Sprintf("Bot %s", discordToken))
	if err != nil {
		return nil, err
	}

	// TODO configurable log level
	d.LogLevel = discordgo.LogInformational

	s, err := client.ConnectNet(ctx, routerURL, cfg)
	if err != nil {
		return nil, fmt.Errorf("connection to WAMP router failed: %s", err)
	}

	return NewComponent(d, s), nil
}

// Open opens the component.
func (c *Component) Open() error {
	err := c.discordSess.Open()
	if err != nil {
		return fmt.Errorf("connection to discord failed: %s", err)
	}

	c.addHandlers()
	return c.registerProcedures()
}

// Closes the component.
func (c *Component) Close() error {
	return joinErrors(
		c.discordSess.Close(),
		c.wampClient.Close(),
	)
}

// Done returns a channel which is closed when the component is done
func (c *Component) Done() <-chan struct{} {
	return c.wampClient.Done()
}

func (c *Component) addHandlers() {
	c.discordSess.AddHandler(func(s *discordgo.Session, u *discordgo.VoiceStateUpdate) {
		_ = c.wampClient.Publish("com.discord.on_voice_state_update", nil, wamp.List{u}, nil)
	})

	c.discordSess.AddHandler(func(s *discordgo.Session, u *discordgo.VoiceServerUpdate) {
		_ = c.wampClient.Publish("com.discord.on_voice_server_update", nil, wamp.List{u}, nil)
	})
}

func (c *Component) registerProcedures() error {
	return joinErrors(
		c.wampClient.Register("com.discord.meta.assert_ready", c.assertReady, nil),
		c.wampClient.Register("com.discord.update_voice_state", c.updateVoiceState, nil),
	)
}

func (c *Component) assertReady(ctx context.Context, invocation *wamp.Invocation) client.InvokeResult {
	if !c.wampClient.Connected() {
		return client.InvokeResult{
			Err:  discordErrURI,
			Args: []interface{}{"not connected to discord"},
		}
	}

	return client.InvokeResult{}
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

	err := c.discordSess.ChannelVoiceJoinManual(gID, cID, mute, deaf)

	if err != nil {
		return resultFromErrorURI(discordErrURI, err.Error())
	}

	return emptyResult
}

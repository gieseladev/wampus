// package wampus provides a WAMP component for Discord.
package wampus

import (
	"context"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/gammazero/nexus/client"
	"github.com/gammazero/nexus/wamp"
	"strings"
)

var (
	emptyResult = &client.InvokeResult{}
)

func resultFromError(err error) *client.InvokeResult {
	return &client.InvokeResult{
		Err: wamp.URI(err.Error()),
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
func Connect(discordToken string, routerURL string, cfg client.Config) (*Component, error) {
	d, err := discordgo.New(fmt.Sprintf("Bot %s", discordToken))
	if err != nil {
		return nil, err
	}

	s, err := client.ConnectNet(routerURL, cfg)
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

	return c.registerProcedures()
}

// Closes the component.
func (c *Component) Close() error {
	return joinErrors(
		c.discordSess.Close(),
		c.wampClient.Close(),
	)
}

func (c *Component) addHandlers() {
	c.discordSess.AddHandler(func(s *discordgo.Session, u *discordgo.VoiceStateUpdate) {
		_ = c.wampClient.Publish("com.discord.voice_state_update", nil, wamp.List{u}, nil)
	})

	c.discordSess.AddHandler(func(s *discordgo.Session, u *discordgo.VoiceServerUpdate) {
		_ = c.wampClient.Publish("com.discord.voice_server_update", nil, wamp.List{u}, nil)
	})
}

func (c *Component) registerProcedures() error {
	return joinErrors(
		c.wampClient.Register("com.discord.update_voice_state", c.updateVoiceState, nil),
	)
}

func (c *Component) updateVoiceState(ctx context.Context, args wamp.List, kwargs, details wamp.Dict) *client.InvokeResult {
	gID, _ := wamp.AsString(args[0])
	cID, _ := wamp.AsString(args[1])

	mute, _ := wamp.AsBool(kwargs["mute"])
	deaf, _ := wamp.AsBool(kwargs["deaf"])

	// TODO does this handle disconnects properly?
	err := c.discordSess.ChannelVoiceJoinManual(gID, cID, mute, deaf)
	if err != nil {
		return resultFromError(err)
	}

	return emptyResult
}

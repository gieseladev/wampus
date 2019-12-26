// package wampus provides a WAMP component for Discord.
package wampus

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/gammazero/nexus/v3/client"
	"github.com/gammazero/nexus/v3/wamp"
)

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
	d.LogLevel = discordgo.LogDebug

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
		// Meta
		c.wampClient.Register(DiscordURIPrefix+"meta.assert_ready", c.assertReady, nil),
		c.registerGatewayProcedures(),
		c.registerTokenProcedures(),
	)
}

func (c *Component) assertReady(_ context.Context, _ *wamp.Invocation) client.InvokeResult {
	if !c.wampClient.Connected() {
		return client.InvokeResult{
			Err:  ErrURI,
			Args: []interface{}{"not connected to discord"},
		}
	}

	return client.InvokeResult{}
}

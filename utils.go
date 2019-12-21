package wampus

import (
	"context"
	"errors"
	"github.com/bwmarrin/discordgo"
	"github.com/gammazero/nexus/v3/client"
	"github.com/gammazero/nexus/v3/wamp"
	"net/http"
	"strconv"
	"strings"
)

// raceContext races a context against a channel.
// If the context completes faster than the channel, the context's error is
// returned.
// In case the channel completes faster than the context nil is returned.
func raceContext(ctx context.Context, done <-chan struct{}) error {
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// fnRaceContext races a function against a context.
// If the context completes first, the context's error is returned.
// Otherwise, the result of the function is returned.
//
// This function primarily serves to make it possible to silently abort blocking
// functions without context support.
func fnRaceContext(ctx context.Context, fn func() error) error {
	var fErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		fErr = fn()
	}()

	if err := raceContext(ctx, done); err != nil {
		return err
	}

	return fErr
}

// emptyResult represents a completely empty invoke result.
var emptyResult = client.InvokeResult{}

// resultFromErrorURI creates an invoke result with the given error uri.
// Additional arguments are passed to the result's arguments.
func resultFromErrorURI(uri wamp.URI, args ...interface{}) client.InvokeResult {
	return client.InvokeResult{
		Err:  uri,
		Args: args,
	}
}

// resultFromRESTError creates an invoke result from a discord rest error.
func resultFromRESTError(err discordgo.RESTError) client.InvokeResult {
	var code int
	if err.Message == nil {
		code = err.Response.StatusCode
	} else {
		code = err.Message.Code
	}

	var res client.InvokeResult
	switch code {
	case http.StatusUnauthorized:
		res = resultFromErrorURI(UnauthorizedURI)
	case http.StatusNotFound:
		res = resultFromErrorURI(NotFoundURI)
	default:
		res = resultFromErrorURI(ErrURI, "unexpected error response", err.Error())
	}

	if res.Kwargs == nil {
		res.Kwargs = wamp.Dict{}
	}

	res.Kwargs["status_code"] = err.Response.StatusCode

	return res
}

// resultFromDiscordErr creates a invoke result from an error returned by a
// discordgo function.
// Passing a nil error will cause the function to panic!
func resultFromDiscordErr(err error) client.InvokeResult {
	if err == nil {
		panic("passed nil error")
	}
	switch err := err.(type) {
	case discordgo.RESTError:
		return resultFromRESTError(err)
	default:
		return resultFromErrorURI(ErrURI, "unknown error", err.Error())
	}
}

// joinErrors combines all non-nil errors into a single error.
// If no non-nil error is passed, the result is nil.
// If exactly one non-nil error is passed, the result is that error.
func joinErrors(errs ...error) error {
	var lastErr error
	var errStrings []string

	for _, err := range errs {
		if err != nil {
			lastErr = err
			errStrings = append(errStrings, err.Error())
		}
	}

	switch len(errStrings) {
	case 0:
		return nil
	case 1:
		return lastErr
	default:
		return errors.New(strings.Join(errStrings, "\n"))
	}
}

// asSnowflake is an extended type assertion which handles Twitter snowflakes.
// Either integers or strings are accepted.
func asSnowflake(v interface{}) (string, bool) {
	if s, ok := wamp.AsString(v); ok {
		return s, true
	} else if i, ok := wamp.AsInt64(v); ok {
		return strconv.FormatInt(i, 10), true
	} else {
		return "", false
	}
}

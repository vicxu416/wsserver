package middleware

import (
	"github.com/labstack/gommon/random"
	"github.com/vicxu416/wsserver"
)

// RequestIDConfig defines the config for RequestID middleware
type RequestIDConfig struct {
	Generator func() string
}

var (
	// DefaultRequestIDConfig is the default RequestID middleware config.
	DefaultRequestIDConfig = RequestIDConfig{
		Generator: generator,
	}
)

// RequestID returns a X-Request-ID middleware.
func RequestID() wsserver.MiddlewareFunc {
	return RequestIDWithConfig(DefaultRequestIDConfig)
}

// RequestIDWithConfig returns a X-Request-ID middleware with config.
func RequestIDWithConfig(config RequestIDConfig) wsserver.MiddlewareFunc {
	// Defaults

	if config.Generator == nil {
		config.Generator = generator
	}

	return func(handle wsserver.MsgHandlerFunc) wsserver.MsgHandlerFunc {
		return func(c *wsserver.Context) error {
			rid := config.Generator()
			c.Set(wsserver.CtxRequestID, rid)
			err := handle(c)
			return err
		}
	}
}

func generator() string {
	return random.String(32)
}

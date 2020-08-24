package middleware

import (
	"fmt"
	"runtime"

	"github.com/vicxu416/wsserver"
)

// RecoveryConfig defines the config for Recovery middleware
type RecoveryConfig struct {
	StackSize         int
	DisableStackAll   bool
	DisablePrintStack bool
}

// DefaultRecoverConfig is the default Recover middleware config.
var DefaultRecoverConfig = RecoveryConfig{
	StackSize:         4 << 10, // 4 KB
	DisableStackAll:   false,
	DisablePrintStack: false,
}

// Recover returns a middleware which recovers from panics anywhere in the chain
// and handles the control to the centralized HTTPErrorHandler.
func Recover() wsserver.MiddlewareFunc {
	return RecoverWithConfig(DefaultRecoverConfig)
}

// RecoverWithConfig returns a Recover middleware with config.
func RecoverWithConfig(config RecoveryConfig) wsserver.MiddlewareFunc {
	return func(handle wsserver.MsgHandlerFunc) wsserver.MsgHandlerFunc {
		return func(c *wsserver.Context) error {
			defer func() {
				if err := recover(); err != nil {
					err, ok := err.(error)
					if !ok {
						err = fmt.Errorf("%+v", err)
					}
					stack := make([]byte, config.StackSize)
					length := runtime.Stack(stack, config.DisableStackAll)
					if !config.DisablePrintStack {
						msg := fmt.Sprintf("[PANIC RECOVER] %v %s\n", err, stack[:length])
						c.Logger().Debug(msg)
					}
					c.Error(err)
				}
			}()

			err := handle(c)
			return err
		}
	}
}

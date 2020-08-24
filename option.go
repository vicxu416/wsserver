package wsserver

// ConnErrHandle handle error and connection context
type ConnErrHandle func(*Context, error)

// ConnHandle handle connection context
type ConnHandle func(*Context)

// Option option function
type Option func(o *Options)

// Options webscoket option
type Options struct {
	ConnOpendHook  ConnHandle
	ConnClosedHook ConnHandle
	Logger         Logger
}

// ConnHooks represetn connection opened hook and closed hook
func ConnHooks(opened, closed ConnHandle) Option {
	return func(o *Options) {
		o.ConnOpendHook = opened
		o.ConnClosedHook = closed
	}
}

// SetLogger set wssever logger
func SetLogger(logger Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// DefaultOptions default options
var DefaultOptions = Options{
	Logger:         newDefualtLogger(),
	ConnOpendHook:  defaultConnOpenHook,
	ConnClosedHook: defaultConnCloseHook,
}

func defaultConnOpenHook(ctx *Context) {
	ctx.Logger().Infof("ws: websocket connection was built, connection id:%s", ctx.ID)
}

func defaultConnCloseHook(ctx *Context) {
	ctx.Logger().Infof("ws: websocket connection was closed, connection id:%s", ctx.ID)
}

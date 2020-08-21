package wsserver

// ErrorHandle handle websocket connection context error
type ErrorHandle func(error)

// ConnErrHandle handle error and connection context
type ConnErrHandle func(*Context, error)

// ConnHandle handle connection context
type ConnHandle func(*Context)

// Option option function
type Option func(o *Options)

// Options webscoket option
type Options struct {
	ErrHandler     ErrorHandle
	ConnOpendHook  ConnHandle
	ConnClosedHook ConnHandle
}

// WSErrorHandle websocket error handle
func WSErrorHandle(handle func(err error)) Option {
	return func(o *Options) {
		o.ErrHandler = handle
	}
}

// ConnHooks represetn connection opened hook and closed hook
func ConnHooks(opened, closed func(ctx *Context)) Option {
	return func(o *Options) {
		o.ConnOpendHook = opened
		o.ConnOpendHook = closed
	}
}

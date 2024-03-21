package eventbus

type ProxyCreator func(bus Eventbus) Eventbus

type (
	eventbusOptions struct {
		proxyCreators []ProxyCreator
	}

	EventbusOption interface {
		apply(*eventbusOptions)
	}
)

type funcEventbusOption struct {
	f func(options *eventbusOptions)
}

func (fdo *funcEventbusOption) apply(do *eventbusOptions) {
	fdo.f(do)
}

func newFuncEventbusOption(f func(*eventbusOptions)) *funcEventbusOption {
	return &funcEventbusOption{
		f: f,
	}
}

// WithProxyCreator returns a EventbusOption that sets the ProxyCreator.
func WithProxys(proxyCreators ...ProxyCreator) EventbusOption {
	return newFuncEventbusOption(func(o *eventbusOptions) {
		o.proxyCreators = proxyCreators
	})
}

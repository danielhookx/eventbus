package eventbus

import (
	"context"
	"fmt"
	"reflect"

	"github.com/danielhookx/fission"
)

func Func(fn reflect.Value) context.Context {
	return context.WithValue(context.Background(), "eventbus", fn)
}

func FromCtx(ctx context.Context) reflect.Value {
	fn := ctx.Value("eventbus")
	return fn.(reflect.Value)
}

type BusSubscriber interface {
	Subscribe(topic string, fn interface{}) error
	SubscribeSync(topic string, fn interface{}) error
	Unsubscribe(topic string, handler interface{}) error
}

type BusPublisher interface {
	Publish(topic string, args ...interface{})
}

type Eventbus interface {
	BusSubscriber
	BusPublisher
}

type EventBus struct {
	cm *fission.CenterManager
	dm *fission.DistributorManager
}

func NewEventBus(opt ...EventbusOption) Eventbus {
	opts := eventbusOptions{}
	for _, o := range opt {
		o.apply(&opts)
	}
	var bus Eventbus
	bus = &EventBus{
		cm: fission.NewCenterManager(),
		dm: fission.NewDistributorManager(),
	}
	for _, proxyCreator := range opts.proxyCreators {
		bus = proxyCreator(bus)
	}
	return bus
}

func (bus *EventBus) CreateEventBusSyncDist(ctx context.Context, key any) fission.Distribution {
	fn := FromCtx(ctx)
	return NewSyncDistribution(fn)
}

func (bus *EventBus) CreateEventBusAsyncDist(ctx context.Context, key any) fission.Distribution {
	fn := FromCtx(ctx)
	return NewAsyncDistribution(fn)
}

// Wrapper function that transforms a function into a comparable interface.
func functionWrapper(f interface{}) interface{} {
	return reflect.ValueOf(f).Pointer()
}

func (bus *EventBus) Subscribe(topic string, fn interface{}) error {
	fnType := reflect.TypeOf(fn)
	if !(fnType.Kind() == reflect.Func) {
		return fmt.Errorf("%s is not of type reflect.Func", fnType.Kind())
	}

	r := bus.cm.PutCenter(topic)
	p := bus.dm.PutDistributor(Func(reflect.ValueOf(fn)), functionWrapper(fn), bus.CreateEventBusAsyncDist)
	r.AddDistributor(p)
	return nil
}

func (bus *EventBus) SubscribeSync(topic string, fn interface{}) error {
	fnType := reflect.TypeOf(fn)
	if !(fnType.Kind() == reflect.Func) {
		return fmt.Errorf("%s is not of type reflect.Func", fnType.Kind())
	}

	r := bus.cm.PutCenter(topic)
	p := bus.dm.PutDistributor(Func(reflect.ValueOf(fn)), functionWrapper(fn), bus.CreateEventBusSyncDist)
	r.AddDistributor(p)
	return nil
}

func (bus *EventBus) Unsubscribe(topic string, handler interface{}) error {
	fnType := reflect.TypeOf(handler)
	if !(fnType.Kind() == reflect.Func) {
		return fmt.Errorf("%s is not of type reflect.Func", fnType.Kind())
	}
	r := bus.cm.PutCenter(topic)
	r.DelDistributor(functionWrapper(handler))
	return nil
}

func (bus *EventBus) Publish(topic string, args ...interface{}) {
	r := bus.cm.PutCenter(topic)
	r.Fission(args)
	return
}

type SyncDistribution struct {
	fn reflect.Value
}

func NewSyncDistribution(fn reflect.Value) *SyncDistribution {
	return &SyncDistribution{
		fn: fn,
	}
}

func (d *SyncDistribution) Dist(data any) error {
	passedArguments := setFuncArgs(d.fn, data.([]interface{}))
	d.fn.Call(passedArguments)
	return nil
}

func (d *SyncDistribution) Close() error {
	return nil
}

type AsyncDistribution struct {
	fn reflect.Value
}

func NewAsyncDistribution(fn reflect.Value) *AsyncDistribution {
	return &AsyncDistribution{
		fn: fn,
	}
}

func (d *AsyncDistribution) Dist(data any) error {
	go func() {
		passedArguments := setFuncArgs(d.fn, data.([]interface{}))
		d.fn.Call(passedArguments)
	}()
	return nil
}

func (d *AsyncDistribution) Close() error {
	return nil
}

func setFuncArgs(fn reflect.Value, args []interface{}) []reflect.Value {
	funcType := fn.Type()
	passedArguments := make([]reflect.Value, len(args))
	for i, v := range args {
		if v == nil {
			passedArguments[i] = reflect.New(funcType.In(i)).Elem()
		} else {
			passedArguments[i] = reflect.ValueOf(v)
		}
	}
	return passedArguments
}

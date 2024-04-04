package eventbus

import (
	"context"
	"fmt"
	"reflect"
	"sync"

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
	SubscribeWith(topic string, key any, fn any, distHandler fission.CreateDistributionHandleFunc) error
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

func CreateEventBusSyncDist(ctx context.Context, key any) fission.Distribution {
	fn := FromCtx(ctx)
	return NewSyncDistribution(fn)
}

func CreateEventBusAsyncDist(ctx context.Context, key any) fission.Distribution {
	fn := FromCtx(ctx)
	return NewAsyncDistribution(fn)
}

func CreateEventBusRepeatDist(ctx context.Context, key any) fission.Distribution {
	return NewRepeatDistribution()
}

// Wrapper function that transforms a function into a comparable interface.
func functionWrapper(f interface{}) uintptr {
	return reflect.ValueOf(f).Pointer()
}

func (bus *EventBus) Subscribe(topic string, fn interface{}) error {
	fnType := reflect.TypeOf(fn)
	if !(fnType.Kind() == reflect.Func) {
		return fmt.Errorf("%s is not of type reflect.Func", fnType.Kind())
	}

	handler := reflect.ValueOf(fn)
	key := functionWrapper(fn)
	r := bus.cm.PutCenter(topic)
	p := bus.dm.PutDistributor(Func(handler), key, CreateEventBusRepeatDist)
	rd := p.(*RepeatDistribution)
	rd.Add(NewAsyncDistribution(reflect.ValueOf(fn)))
	r.AddDistributor(key, p)
	return nil
}

func (bus *EventBus) SubscribeSync(topic string, fn interface{}) error {
	fnType := reflect.TypeOf(fn)
	if !(fnType.Kind() == reflect.Func) {
		return fmt.Errorf("%s is not of type reflect.Func", fnType.Kind())
	}

	handler := reflect.ValueOf(fn)
	key := functionWrapper(fn)
	r := bus.cm.PutCenter(topic)
	p := bus.dm.PutDistributor(Func(handler), key, CreateEventBusRepeatDist)
	rd := p.(*RepeatDistribution)
	rd.Add(NewSyncDistribution(reflect.ValueOf(fn)))
	r.AddDistributor(key, p)
	return nil
}

func (bus *EventBus) SubscribeWith(topic string, key any, fn any, distHandler fission.CreateDistributionHandleFunc) error {
	fnType := reflect.TypeOf(fn)
	if !(fnType.Kind() == reflect.Func) {
		return fmt.Errorf("%s is not of type reflect.Func", fnType.Kind())
	}

	if key == nil {
		return fmt.Errorf("key is nil")
	}

	r := bus.cm.PutCenter(topic)
	p := bus.dm.PutDistributor(Func(reflect.ValueOf(fn)), key, distHandler)
	r.AddDistributor(key, p)
	return nil
}

func (bus *EventBus) Unsubscribe(topic string, handler interface{}) error {
	fnType := reflect.TypeOf(handler)
	if fnType.Kind() == reflect.Func {
		r := bus.cm.PutCenter(topic)
		r.DelDistributor(functionWrapper(handler))
		return nil
	}
	if fnType.Kind() == reflect.Uintptr {
		r := bus.cm.PutCenter(topic)
		r.DelDistributor(handler)
		return nil
	}
	return fmt.Errorf("Unsubscribe invalid handler type %s", fnType.Kind())
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

type RepeatDistribution struct {
	lock  sync.RWMutex
	dists []fission.Distribution
}

func NewRepeatDistribution() *RepeatDistribution {
	return &RepeatDistribution{
		dists: []fission.Distribution{},
	}
}

func (d *RepeatDistribution) Add(dist fission.Distribution) error {
	d.lock.Lock()
	d.dists = append(d.dists, dist)
	d.lock.Unlock()
	return nil
}

func (d *RepeatDistribution) Dist(data any) error {
	d.lock.RLock()
	dists := make([]fission.Distribution, len(d.dists))
	copy(dists, d.dists)
	d.lock.RUnlock()
	for _, dist := range dists {
		dist.Dist(data)
	}
	return nil
}

func (d *RepeatDistribution) Close() error {
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

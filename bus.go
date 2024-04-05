package eventbus

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/danielhookx/fission"
)

type BusSubscriber interface {
	Subscribe(topic string, fn interface{}) error
	SubscribeSync(topic string, fn interface{}) error
	Unsubscribe(topic string, key any) error
	SubscribeWith(topic string, key any, distHandler fission.CreateDistributionHandleFunc) error
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

func (bus *EventBus) Subscribe(topic string, fn interface{}) error {
	fnType := reflect.TypeOf(fn)
	if !(fnType.Kind() == reflect.Func) {
		return fmt.Errorf("%s is not of type reflect.Func", fnType.Kind())
	}

	handler := reflect.ValueOf(fn)
	key := handler.Pointer()

	r := bus.cm.PutCenter(topic)
	p := bus.dm.PutDistributor(key, CreateEventBusRepeatDist)
	p.Register(toDistCtx(NewAsyncDistribution(handler)))
	r.AddDistributor(p)
	return nil
}

func (bus *EventBus) SubscribeSync(topic string, fn interface{}) error {
	fnType := reflect.TypeOf(fn)
	if !(fnType.Kind() == reflect.Func) {
		return fmt.Errorf("%s is not of type reflect.Func", fnType.Kind())
	}

	handler := reflect.ValueOf(fn)
	key := handler.Pointer()

	r := bus.cm.PutCenter(topic)
	p := bus.dm.PutDistributor(key, CreateEventBusRepeatDist)
	p.Register(toDistCtx(NewSyncDistribution(handler)))
	r.AddDistributor(p)
	return nil
}

func (bus *EventBus) SubscribeWith(topic string, key any, distHandler fission.CreateDistributionHandleFunc) error {
	if key == nil {
		return fmt.Errorf("key is nil")
	}

	r := bus.cm.PutCenter(topic)
	p := bus.dm.PutDistributor(key, distHandler)
	p.Register(nil)
	r.AddDistributor(p)
	return nil
}

func (bus *EventBus) Unsubscribe(topic string, key any) error {
	fnType := reflect.TypeOf(key)
	if fnType.Kind() == reflect.Func {
		r := bus.cm.PutCenter(topic)
		r.DelDistributor(reflect.ValueOf(key).Pointer())
		return nil
	}
	r := bus.cm.PutCenter(topic)
	r.DelDistributor(key)
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

func (d *SyncDistribution) Register(ctx context.Context) {
	return
}

func (d *SyncDistribution) Key() any {
	return d.fn.Pointer()
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

func (d *AsyncDistribution) Register(ctx context.Context) {
	return
}

func (d *AsyncDistribution) Key() any {
	return d.fn.Pointer()
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
	key   any
	lock  sync.RWMutex
	dists []fission.Distribution
}

func NewRepeatDistribution(key any) *RepeatDistribution {
	return &RepeatDistribution{
		key:   key,
		dists: []fission.Distribution{},
	}
}

func (d *RepeatDistribution) Register(ctx context.Context) {
	d.lock.Lock()
	dist := fromDistCtx(ctx)
	d.dists = append(d.dists, dist)
	d.lock.Unlock()
}

func (d *RepeatDistribution) Key() any {
	return d.key
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

func toDistCtx(dist fission.Distribution) context.Context {
	return context.WithValue(context.Background(), "eventbus", dist)
}

func fromDistCtx(ctx context.Context) fission.Distribution {
	dist := ctx.Value("eventbus")
	return dist.(fission.Distribution)
}

func CreateEventBusRepeatDist(key any) fission.Distribution {
	return NewRepeatDistribution(key)
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

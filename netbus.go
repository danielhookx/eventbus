package eventbus

import (
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/url"

	"github.com/danielhookx/fission"
)

type SubArgs struct {
	RemoteURL string
	Topic     string
}

type SubReply struct {
}

type UnsubArgs struct {
	Topic string
}

type UnsubReply struct {
}

type PubArgs struct {
	Topic string
	Data  any
}

type PubReply struct{}

type RPCProxy struct {
	rawURL    string
	remoteURL string
	bus       Eventbus
}

func NewRPCProxyCreator(rawURL, remoteURL string) ProxyCreator {
	return func(bus Eventbus) Eventbus {
		b, err := NewRPCProxy(rawURL, remoteURL, bus)
		if err != nil {
			panic(err)
		}
		return b
	}
}

func NewRPCProxy(rawURL, remoteURL string, bus Eventbus) (*RPCProxy, error) {
	p := &RPCProxy{
		rawURL:    rawURL,
		remoteURL: remoteURL,
		bus:       bus,
	}
	gob.Register([]interface{}{})
	rpc.Register(p)
	// Registers an HTTP handler for RPC messages
	// rpc.HandleHTTP()

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen(u.Scheme, parseAddress(u))
	if err != nil {
		return nil, err
	}
	// start http server goroutine
	// go http.Serve(listener, nil)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Fatal("Accept error:", err)
			}
			go rpc.ServeConn(conn)
		}
	}()
	return p, nil
}

func (p *RPCProxy) Subscribe(topic string, fn interface{}) error {
	// Call the remote subscription method to register the event to the remote endpoint.
	remote, err := url.Parse(p.remoteURL)
	if err != nil {
		return fmt.Errorf("Parse remote url error: %w", err)
	}
	client, err := rpc.Dial(remote.Scheme, parseAddress(remote))
	if err != nil {
		return fmt.Errorf("Client connection error %w", err)
	}

	reply := SubReply{}
	err = client.Call("RPCProxy.RPCSubscribe", &SubArgs{
		RemoteURL: p.rawURL,
		Topic:     topic,
	}, &reply)
	if err != nil {
		return fmt.Errorf("Client invocation error: %w", err)
	}
	// Call the local subscription method and register the callback locally
	return p.bus.Subscribe(topic, fn)
}

func (p *RPCProxy) SubscribeSync(topic string, fn interface{}) error {
	// Call the remote subscription method to register the event to the remote endpoint.
	remote, err := url.Parse(p.remoteURL)
	if err != nil {
		return fmt.Errorf("Parse remote url error: %w", err)
	}
	client, err := rpc.Dial(remote.Scheme, parseAddress(remote))
	if err != nil {
		return fmt.Errorf("Client connection error %w", err)
	}

	reply := &SubReply{}
	err = client.Call("RPCProxy.RPCSubscribeSync", &SubArgs{
		RemoteURL: p.rawURL,
		Topic:     topic,
	}, &reply)
	if err != nil {
		return fmt.Errorf("Client invocation error: %w", err)
	}
	// Call the local subscription method and register the callback locally
	return p.bus.SubscribeSync(topic, fn)
}

func (p *RPCProxy) SubscribeWith(topic string, key any, distHandler fission.CreateDistributionHandleFunc) error {
	return p.bus.SubscribeWith(topic, key, distHandler)
}

func (p *RPCProxy) Unsubscribe(topic string, handler interface{}) error {
	// Call the remote unsubscription method to remove the event.
	remote, err := url.Parse(p.remoteURL)
	if err != nil {
		return fmt.Errorf("Parse remote url error: %w", err)
	}
	client, err := rpc.Dial(remote.Scheme, parseAddress(remote))
	if err != nil {
		return fmt.Errorf("Client connection error %w", err)
	}

	reply := &UnsubReply{}
	err = client.Call("RPCProxy.RPCUnsubscribe", &UnsubArgs{
		Topic: topic,
	}, &reply)
	if err != nil {
		return fmt.Errorf("Client invocation error: %w", err)
	}
	// Call the local unsubscription method and remove the callback locally
	return p.bus.Unsubscribe(topic, handler)
}

func (p *RPCProxy) Publish(topic string, args ...interface{}) {
	p.bus.Publish(topic, args...)
}

func (p *RPCProxy) RPCSubscribe(args *SubArgs, reply *SubReply) error {
	// Receive subscription method calls from the peer
	// callback method actually executes the remote call of Publish
	return p.bus.SubscribeWith(args.Topic, args.Topic, createNetPublishDist(args))
}

func (p *RPCProxy) RPCSubscribeSync(args *SubArgs, reply *SubReply) error {
	// Receive subscription method calls from the peer
	// callback method actually executes the remote call of Publish
	p.bus.SubscribeWith(args.Topic, args.Topic, createNetPublishDist(args))
	return nil
}

func (p *RPCProxy) RPCUnsubscribe(args *UnsubArgs, reply *UnsubReply) error {
	p.bus.Unsubscribe(args.Topic, args.Topic)
	return nil
}

func (p *RPCProxy) RPCPublish(args *PubArgs, reply *PubReply) error {
	params := args.Data.([]interface{})
	p.bus.Publish(args.Topic, params...)
	return nil
}

func createNetPublishDist(args *SubArgs) fission.CreateDistributionHandleFunc {
	return func(key any) fission.Distribution {
		return &netPublishDist{
			key:  key,
			args: args,
		}
	}
}

type netPublishDist struct {
	key  any
	args *SubArgs
}

func (d *netPublishDist) Register(ctx context.Context) {
	return
}

func (d *netPublishDist) Key() any {
	return d.key
}

func (d *netPublishDist) Dist(data any) error {
	remote, err := url.Parse(d.args.RemoteURL)
	if err != nil {
		return fmt.Errorf("Parse remote url error: %w", err)
	}
	client, err := rpc.Dial(remote.Scheme, parseAddress(remote))
	if err != nil {
		return fmt.Errorf("Client connection error %w", err)
	}

	reply := &PubReply{}
	err = client.Call("RPCProxy.RPCPublish", &PubArgs{
		Topic: d.args.Topic,
		Data:  data,
	}, &reply)
	if err != nil {
		return fmt.Errorf("Client invocation error: %w", err)
	}
	return nil
}

func (d *netPublishDist) Close() error {
	return nil
}

func parseAddress(u *url.URL) string {
	if u.Scheme == "unix" {
		return u.Path
	}
	return u.Host
}

package eventbus

import (
	"encoding/gob"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"net/url"
)

type SubArgs struct {
	RemoteURL string
	Topic     string
}

type SubReply struct {
}

type PubArgs struct {
	Topic string
	Data  any
}

type PubReply struct {
}

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
	rpc.HandleHTTP()

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen(u.Scheme, fmt.Sprintf(":%s", u.Port()))
	if err != nil {
		return nil, err
	}
	go http.Serve(listener, nil)
	return p, nil
}

func (p *RPCProxy) Subscribe(topic string, fn interface{}) error {
	// Call the remote subscription method to register the event to the remote endpoint.
	remote, err := url.Parse(p.remoteURL)
	if err != nil {
		return fmt.Errorf("Parse remote url error: %w", err)
	}
	client, err := rpc.DialHTTP(remote.Scheme, remote.Host)
	if err != nil {
		return fmt.Errorf("Client connection error %w", err)
	}

	reply := &SubReply{}
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
	client, err := rpc.DialHTTP(remote.Scheme, remote.Host)
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

func (p *RPCProxy) Unsubscribe(topic string, handler interface{}) error {
	// Call the remote subscription method to register the event to the remote endpoint.
	remote, err := url.Parse(p.remoteURL)
	if err != nil {
		return fmt.Errorf("Parse remote url error: %w", err)
	}
	client, err := rpc.DialHTTP(remote.Scheme, remote.Host)
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
	return p.bus.Unsubscribe(topic, handler)
}

func (p *RPCProxy) Publish(topic string, args ...interface{}) {
	p.bus.Publish(topic, args)
}

func (p *RPCProxy) RPCSubscribe(args *SubArgs, reply *SubReply) error {
	// Receive subscription method calls from the peer
	// callback method actually executes the remote call of Publish
	cb := p.doSubscribeCallback(args, reply)
	p.bus.Subscribe(args.Topic, cb)
	return nil
}

func (p *RPCProxy) RPCSubscribeSync(args *SubArgs, reply *SubReply) error {
	// Receive subscription method calls from the peer
	// callback method actually executes the remote call of Publish
	cb := p.doSubscribeCallback(args, reply)
	p.bus.SubscribeSync(args.Topic, cb)
	return nil
}

func (p *RPCProxy) RPCUnsubscribe(args *SubArgs, reply *SubReply) error {
	// Receive subscription method calls from the peer
	// callback method actually executes the remote call of Publish
	cb := p.doSubscribeCallback(args, reply)
	p.bus.Unsubscribe(args.Topic, cb)
	return nil
}

func (p *RPCProxy) RPCPublish(args *PubArgs, reply *PubReply) error {
	params := args.Data.([]interface{})
	p.bus.Publish(args.Topic, params...)
	return nil
}

type subscribeCallback func(data any) error

func (p *RPCProxy) doSubscribeCallback(args *SubArgs, reply *SubReply) subscribeCallback {
	return func(data any) error {
		remote, err := url.Parse(args.RemoteURL)
		if err != nil {
			return fmt.Errorf("Parse remote url error: %w", err)
		}
		client, err := rpc.DialHTTP(remote.Scheme, remote.Host)
		if err != nil {
			return fmt.Errorf("Client connection error %w", err)
		}

		reply := &PubReply{}
		err = client.Call("RPCProxy.RPCPublish", &PubArgs{
			Topic: args.Topic,
			Data:  data,
		}, &reply)
		if err != nil {
			return fmt.Errorf("Client invocation error: %w", err)
		}
		return nil
	}
}

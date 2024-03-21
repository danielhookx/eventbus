package eventbus

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"net/url"

	"github.com/google/uuid"
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
	id        string
	rawURL    string
	remoteURL string
	bus       Eventbus
}

func NewRPCProxyCreator(rawURL, remoteURL string) ProxyCreator {
	return func(bus Eventbus) Eventbus {
		return NewRPCProxy(rawURL, remoteURL, bus)
	}
}

func NewRPCProxy(rawURL, remoteURL string, bus Eventbus) *RPCProxy {
	p := &RPCProxy{
		id:        uuid.New().String(),
		rawURL:    rawURL,
		remoteURL: remoteURL,
		bus:       bus,
	}
	rpc.Register(p)
	// Registers an HTTP handler for RPC messages
	rpc.HandleHTTP()

	u, err := url.Parse(rawURL)
	if err != nil {
		log.Fatal("parse url error: ", err)
	}

	listener, err := net.Listen(u.Scheme, fmt.Sprintf(":%s", u.Port()))
	if err != nil {
		log.Fatal("Listener error: ", err)
	}
	// Serve accepts incoming HTTP connections on the listener l, creating
	// a new service goroutine for each. The service goroutines read requests
	// and then call handler to reply to them
	go http.Serve(listener, nil)
	return p
}

func (p *RPCProxy) Subscribe(topic string, fn interface{}) error {
	// DialHTTP connects to an HTTP RPC server at the specified network
	remote, err := url.Parse(p.remoteURL)
	if err != nil {
		log.Fatal("parse url error: ", err)
		return err
	}
	client, err := rpc.DialHTTP(remote.Scheme, remote.Host)
	if err != nil {
		log.Fatal("Client connection error: ", err)
		return err
	}

	reply := &SubReply{}
	err = client.Call("RPCProxy.RPCSubscribe", &SubArgs{
		RemoteURL: p.rawURL,
		Topic:     topic,
	}, &reply)
	if err != nil {
		log.Fatal("Client invocation error: ", err)
		return err
	}
	return p.bus.Subscribe(topic, fn)
}

func (p *RPCProxy) SubscribeSync(topic string, fn interface{}) error {
	return nil
}

func (p *RPCProxy) Unsubscribe(topic string, handler interface{}) error {
	return nil
}

func (p *RPCProxy) Publish(topic string, args ...interface{}) {
	p.bus.Publish(topic, args)
}

func (p *RPCProxy) RPCSubscribe(args *SubArgs, reply *SubReply) error {
	p.bus.Subscribe(args.Topic, func(data any) error {
		remote, err := url.Parse(args.RemoteURL)
		if err != nil {
			log.Fatal("parse url error: ", err)
			return err
		}
		client, err := rpc.DialHTTP(remote.Scheme, remote.Host)
		if err != nil {
			log.Fatal("Client connection error: ", err)
			return err
		}

		reply := &PubReply{}
		err = client.Call("RPCProxy.RPCPublish", &PubArgs{
			Topic: args.Topic,
			Data:  data,
		}, &reply)
		if err != nil {
			log.Fatal("Client invocation error: ", err)
			return err
		}
		return nil
	})
	return nil
}

func (p *RPCProxy) RPCPublish(args *PubArgs, reply *PubReply) error {
	p.bus.Publish(args.Topic, args.Data)
	return nil
}

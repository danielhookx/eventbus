package main

import (
	"fmt"

	eventbus "github.com/danielhookx/eventbus"
)

func SayHello(name string) {
	fmt.Printf("hello %s\n", name)
}

func main() {
	rawURL := "tcp://:7634"
	remoteURL := "tcp://localhost:7633"

	bus := eventbus.NewEventBus(eventbus.WithProxys(eventbus.NewRPCProxyCreator(rawURL, remoteURL)))
	bus.Subscribe("test-hello", SayHello)

	select {}
}

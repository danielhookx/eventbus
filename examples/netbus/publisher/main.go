package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	eventbus "github.com/danielhookx/eventbus"
)

func main() {
	rawURL := "tcp://:7633"
	remoteURL := "tcp://localhost:7634"

	bus := eventbus.NewEventBus(eventbus.WithProxys(eventbus.NewRPCProxyCreator(rawURL, remoteURL)))

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("please input...")

	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("input with error", err)
			return
		}

		if input == "\n" {
			break
		}
		fmt.Println("your input is:", input)
		bus.Publish("test-hello", strings.TrimRight(input, "\n"))
	}
	fmt.Println("exit")
}

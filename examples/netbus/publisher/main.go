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
	fmt.Println("Note: All input ends with `Enter`, type `exit` return to upper level")

	topic := "test"

	for {
		fmt.Println("Input topic:")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("input with error", err)
			return
		}
		input = strings.TrimRight(input, "\n")
		if input == "exit" {
			break
		}
		topic = input
		if topic == "" {
			fmt.Println("topic can not be empty")
			continue
		}
		fmt.Printf("topic is: %s\n", topic)
		for {
			fmt.Println("Input text content:")
			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("input with error", err)
				return
			}
			input = strings.TrimRight(input, "\n")
			if input == "exit" {
				break
			}
			bus.Publish(topic, input)
		}
	}
	fmt.Println("exit")
}

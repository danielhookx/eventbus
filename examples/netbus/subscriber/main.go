package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	eventbus "github.com/danielhookx/eventbus"
)

func SayHello(name string) {
	fmt.Printf("hello %s\n", name)
}

func SayHiDely(name string) {
	time.Sleep(1 * time.Second * 2)
	fmt.Printf("hi %s\n", name)
}

func main() {
	rawURL := "tcp://:7634"
	remoteURL := "tcp://localhost:7633"

	bus := eventbus.New(eventbus.WithProxys(eventbus.NewRPCProxyCreator(rawURL, remoteURL)))

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
			fmt.Println(`choose order type:
			1. subscribe
			2. sync subscribe
			3. unsubscribe
			`)
			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("input with error", err)
				return
			}
			commandTypeStr := strings.TrimRight(input, "\n")
			if commandTypeStr == "exit" {
				break
			}

			command, err := strconv.Atoi(commandTypeStr)
			if err != nil {
				fmt.Printf("unsupported command type %v\n", commandTypeStr)
				continue
			}

			fmt.Println(`choose method type:
			1. SayHello
			2. SayHiDely
			`)
			input, err = reader.ReadString('\n')
			if err != nil {
				fmt.Println("input with error", err)
				return
			}
			methodStr := strings.TrimRight(input, "\n")

			method, err := strconv.Atoi(methodStr)
			if err != nil {
				fmt.Printf("unsupported method type %v\n", methodStr)
				continue
			}

			var fn func(string)
			switch method {
			case 1:
				fn = SayHello
			case 2:
				fn = SayHiDely
			}

			switch command {
			case 1:
				bus.Subscribe(topic, fn)
			case 2:
				bus.SubscribeSync(topic, fn)
			case 3:
				bus.Unsubscribe(topic, fn)
			}
		}
	}
	fmt.Println("exit")
}

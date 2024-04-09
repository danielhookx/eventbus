# EventBus

EventBus is a lightweight event bus written in Golang, which provides various enhanced features and functionalities.

## Features

- **Easy to use**: EventBus is a tiny library with an API that is super easy to learn. No configuration needed, you can directly create a default instance of EventBus at any position in your code.

- **Cross-process**: You can enable network options during initialization to use a cross-process event bus. 

- **Non-intrusive**: All extensions are implemented without impacting the code of the basic functionality. You only need to enable the corresponding feature options during initialization.

- **Extensible**: The code provides a unified interface for extending functionality, allowing you to customize your own advanced features.

## Installation
Use go get to install this package.
```shell
go get github.com/danielhookx/xcontainer
```

## Getting Started
### Subscribe Topic And Publish
```go
package main

import (
	"github.com/danielhookx/eventbus"
)

func calculator(a int, b int) {
	fmt.Printf("%d\n", a + b)
}

func main() {
    bus := eventbus.New()
    bus.Subscribe("calculator", calculator);
    bus.Publish("calculator", 10, 20);
}
```

### Sync Subscribe Topic
```go
bus.SubscribeSync("calculator", calculator);
```

### Unsubscribe
```go
bus.Unsubscribe("calculator", calculator)
```

### SubscribeWith
There is an advanced usage of this, using the SubscribeWith method to customize the subscription behavior.

It should be noted that when using SubscribeWith, you need to ensure the uniqueness of the key yourself, and the same key will only be subscribed once.

```go
package main

import (
	"github.com/danielhookx/eventbus"
	"github.com/danielhookx/fission"
)

type mockDist struct {
	key string
}

func createMockDistHandlerFunc(key any) fission.Distribution {
	return &mockDist{
		key: key.(string),
	}
}

func (m *mockDist) Register(ctx context.Context) {
	return
}
func (m *mockDist) Key() any {
	return m.key
}
func (m *mockDist) Dist(data any) error {
	// add your code here
	fmt.Println(data)
	return nil
}
func (m *mockDist) Close() error {
	return nil
}

func main() {
	topic := "main.test"
    bus := eventbus.New()
    bus.SubscribeWith(topic, "key1", createMockDistHandlerFunc)
    bus.Publish(topic, "jack")
	bus.Unsubscribe(topic, "key1")
}
```

### Cross Process Events
You only need to specify the use of netbus during initialization. Later, you can use the cross-process eventbus like the local eventbus.

publisher process
```go
package main

import (
	"github.com/danielhookx/eventbus"
)

func main() {
	rawURL := "tcp://:7633"
	remoteURL := "tcp://localhost:7634"

	bus := eventbus.New(
		eventbus.WithProxys(
			eventbus.NewRPCProxyCreator(rawURL, remoteURL),
		),
	)
	
    bus.Publish("test", "jack")
}
```

subscriber process
```go
package main

import (
	"github.com/danielhookx/eventbus"
)

func main() {
	rawURL := "tcp://:7634"
	remoteURL := "tcp://localhost:7633"

	bus := eventbus.New(
		eventbus.WithProxys(
			eventbus.NewRPCProxyCreator(rawURL, remoteURL),
		),
	)

    bus.Subscribe("test", func(name string) {
	    fmt.Printf("hello %s\n", name)
    })
}
```

## Contributing
We welcome contributions to this project. Please submit pull requests with your proposed changes or improvements.
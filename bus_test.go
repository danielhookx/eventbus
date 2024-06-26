package eventbus

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/danielhookx/fission"
)

func a(name string) {
	fmt.Printf("sub1 -- %s\n", name)
}

func b(name string) {
	fmt.Printf("sub2 -- %s\n", name)
}

func c(name string) {
	fmt.Printf("sub3 -- %s\n", name)
}

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
	fmt.Printf("mock dist -- %s\n", data)
	return nil
}
func (m *mockDist) Close() error {
	return nil
}

func BenchmarkSubPub(b *testing.B) {
	e := New()
	topic := "testpub1"
	b.RunParallel(func(pb *testing.PB) {
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			for pb.Next() {
				e.Subscribe(topic, func(name string) {})
			}
		}()

		go func() {
			defer wg.Done()
			for pb.Next() {
				e.Publish(topic, "jack")
			}
		}()

		wg.Wait()
	})
}

func BenchmarkSubPubSync(b *testing.B) {
	e := New()
	topic := "testpub1"
	b.RunParallel(func(pb *testing.PB) {
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			for pb.Next() {
				e.SubscribeSync(topic, func(name string) {})
			}
		}()

		go func() {
			defer wg.Done()
			for pb.Next() {
				e.Publish(topic, "jack")
			}
		}()

		wg.Wait()
	})
}
func TestNamedSubscribe(t *testing.T) {
	e := New()
	topic := "testpub1"
	e.Subscribe(topic, a)
	e.Subscribe(topic, b)
	e.Subscribe(topic, c)
	e.Publish(topic, "jack")
	e.Unsubscribe(topic, b)
	e.Publish(topic, "jack2")
}

func TestNamedSubscribeSync(t *testing.T) {
	e := New()
	topic := "testpub1"
	e.SubscribeSync(topic, a)
	e.SubscribeSync(topic, b)
	e.SubscribeSync(topic, c)
	e.Publish(topic, "jack")
	e.Unsubscribe(topic, b)
	e.Publish(topic, "jack2")
}

func TestAnonymousSubscribe(t *testing.T) {
	e := New()
	topic := "testpub1"
	e.Subscribe(topic, func(name string) {
		fmt.Printf("sub1 -- %s\n", name)
	})
	e.Subscribe(topic, func(name string) {
		fmt.Printf("sub2 -- %s\n", name)
	})
	e.Subscribe(topic, func(name string) {
		fmt.Printf("sub3 -- %s\n", name)
	})
	e.Publish(topic, "jack")
}

func TestAnonymousSubscribeSync(t *testing.T) {
	e := New()
	topic := "testpub1"
	e.SubscribeSync(topic, func(name string) {
		fmt.Printf("sub1 -- %s\n", name)
	})
	e.SubscribeSync(topic, func(name string) {
		fmt.Printf("sub2 -- %s\n", name)
	})
	e.SubscribeSync(topic, func(name string) {
		fmt.Printf("sub3 -- %s\n", name)
	})
	e.Publish(topic, "jack")
}

func TestBlockSubscribe(t *testing.T) {
	e := New()
	topic := "testpub1"
	wg := sync.WaitGroup{}
	wg.Add(3)
	e.Subscribe(topic, func(name string) {
		defer wg.Done()
		fmt.Printf("sub1 -- %s\n", name)
		time.Sleep(time.Millisecond * 200)
	})
	e.Subscribe(topic, func(name string) {
		defer wg.Done()
		fmt.Printf("sub2 -- %s\n", name)
		time.Sleep(time.Millisecond * 200)
	})
	e.Subscribe(topic, func(name string) {
		defer wg.Done()
		fmt.Printf("sub3 -- %s\n", name)
		time.Sleep(time.Millisecond * 200)
	})
	e.Publish(topic, "jack")
	wg.Wait()
}

func TestBlockSubscribeSync(t *testing.T) {
	e := New()
	topic := "testpub1"
	e.SubscribeSync(topic, func(name string) {
		fmt.Printf("sub1 -- %s\n", name)
		time.Sleep(time.Millisecond * 200)
	})
	e.SubscribeSync(topic, func(name string) {
		fmt.Printf("sub2 -- %s\n", name)
		time.Sleep(time.Millisecond * 200)
	})
	e.SubscribeSync(topic, func(name string) {
		fmt.Printf("sub3 -- %s\n", name)
		time.Sleep(time.Millisecond * 200)
	})
	e.Publish(topic, "jack")
}

type testA struct {
	sync.Mutex
	name string
}

func TestSubscribeParamIsolation(t *testing.T) {
	e := New()
	topic := "testpub1"
	var s1 = testA{
		name: "jack",
	}
	e.Subscribe(topic, func(v *testA) {
		v.Lock()
		defer v.Unlock()
		fmt.Printf("sub1 -- %s\n", v.name)
		v.name = "Lee"
	})
	e.Subscribe(topic, func(v *testA) {
		v.Lock()
		defer v.Unlock()
		fmt.Printf("sub2 -- %s\n", v.name)
		v.name = "Danny"
	})
	e.Subscribe(topic, func(v *testA) {
		v.Lock()
		defer v.Unlock()
		fmt.Printf("sub3 -- %s\n", v.name)
		v.name = "Jay"
	})
	e.Publish(topic, &s1)
}

func TestSubscribeSyncParamIsolation(t *testing.T) {
	e := New()
	topic := "testpub1"
	var s1 = testA{
		name: "jack",
	}
	e.SubscribeSync(topic, func(v *testA) {
		fmt.Printf("sub1 -- %s\n", v.name)
		v.name = "Lee"
	})
	e.SubscribeSync(topic, func(v *testA) {
		fmt.Printf("sub2 -- %s\n", v.name)
		v.name = "Danny"
	})
	e.SubscribeSync(topic, func(v *testA) {
		fmt.Printf("sub3 -- %s\n", v.name)
		v.name = "Jay"
	})
	e.Publish(topic, &s1)
}

func TestCallbackSubscribe(t *testing.T) {
	e := New()
	topic := "testpub1"
	var s1 = testA{
		name: "jack",
	}
	e.SubscribeSync(topic, func(v *testA) {
		fmt.Printf("sub1 -- %s\n", v.name)
		e.SubscribeSync(topic, func(v *testA) {
			fmt.Printf("sub2 -- %s\n", v.name)
		})
	})
	e.Publish(topic, &s1)
	e.Publish(topic, &s1)
}

func TestCallbackPublish(t *testing.T) {
	e := New()
	topic := "testpub1"
	topic2 := "testpub2"
	var s1 = testA{
		name: "jack",
	}
	e.SubscribeSync(topic, func(v *testA) {
		fmt.Printf("sub1 -- %s: %s\n", topic, v.name)
		e.Publish(topic2, &s1)
	})
	e.SubscribeSync(topic2, func(v *testA) {
		fmt.Printf("sub2 -- %s: %s\n", topic2, v.name)
	})
	e.Publish(topic, &s1)
}

func TestSubscribeWith(t *testing.T) {
	e := New()
	topic := "testpub1"
	e.SubscribeWith(topic, "key1", createMockDistHandlerFunc)
	e.SubscribeWith(topic, "key2", createMockDistHandlerFunc)
	e.SubscribeWith(topic, "key3", createMockDistHandlerFunc)
	e.Publish(topic, "jack")
}

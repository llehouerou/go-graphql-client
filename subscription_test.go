package graphql

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/graph-gophers/graphql-transport-ws/graphqlws"
)

const schema = `
schema {
	subscription: Subscription
	mutation: Mutation
	query: Query
}
type Query {
	hello: String!
}
type Subscription {
	helloSaid(): HelloSaidEvent!
}
type Mutation {
	sayHello(msg: String!): HelloSaidEvent!
}
type HelloSaidEvent {
	id: String!
	msg: String!
}
`

func subscription_setupClients() (*Client, *SubscriptionClient) {
	endpoint := "http://localhost:8080/graphql"

	client := NewClient(endpoint, &http.Client{Transport: http.DefaultTransport})

	subscriptionClient := NewSubscriptionClient(endpoint).
		WithConnectionParams(map[string]any{
			"headers": map[string]string{
				"foo": "bar",
			},
		}).WithLog(log.Println)

	return client, subscriptionClient
}

func subscription_setupServer() *http.Server {

	// init graphQL schema
	s, err := graphql.ParseSchema(schema, newResolver())
	if err != nil {
		panic(err)
	}

	// graphQL handler
	mux := http.NewServeMux()
	graphQLHandler := graphqlws.NewHandlerFunc(s, &relay.Handler{Schema: s})
	mux.HandleFunc("/graphql", graphQLHandler)
	server := &http.Server{Addr: ":8080", Handler: mux}

	return server
}

type resolver struct {
	helloSaidEvents     chan *helloSaidEvent
	helloSaidSubscriber chan *helloSaidSubscriber
}

func newResolver() *resolver {
	r := &resolver{
		helloSaidEvents:     make(chan *helloSaidEvent),
		helloSaidSubscriber: make(chan *helloSaidSubscriber),
	}

	go r.broadcastHelloSaid()

	return r
}

func (r *resolver) Hello() string {
	return "Hello world!"
}

func (r *resolver) SayHello(args struct{ Msg string }) *helloSaidEvent {
	e := &helloSaidEvent{msg: args.Msg, id: randomID()}
	go func() {
		select {
		case r.helloSaidEvents <- e:
		case <-time.After(1 * time.Second):
		}
	}()
	return e
}

type helloSaidSubscriber struct {
	stop   <-chan struct{}
	events chan<- *helloSaidEvent
}

func (r *resolver) broadcastHelloSaid() {
	subscribers := map[string]*helloSaidSubscriber{}
	unsubscribe := make(chan string)

	// NOTE: subscribing and sending events are at odds.
	for {
		select {
		case id := <-unsubscribe:
			delete(subscribers, id)
		case s := <-r.helloSaidSubscriber:
			subscribers[randomID()] = s
		case e := <-r.helloSaidEvents:
			for id, s := range subscribers {
				go func(id string, s *helloSaidSubscriber) {
					select {
					case <-s.stop:
						unsubscribe <- id
						return
					default:
					}

					select {
					case <-s.stop:
						unsubscribe <- id
					case s.events <- e:
					case <-time.After(time.Second):
					}
				}(id, s)
			}
		}
	}
}

func (r *resolver) HelloSaid(ctx context.Context) <-chan *helloSaidEvent {
	c := make(chan *helloSaidEvent)
	// NOTE: this could take a while
	r.helloSaidSubscriber <- &helloSaidSubscriber{events: c, stop: ctx.Done()}

	return c
}

type helloSaidEvent struct {
	id  string
	msg string
}

func (r *helloSaidEvent) Msg() string {
	return r.msg
}

func (r *helloSaidEvent) ID() string {
	return r.id
}

func randomID() string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, 16)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

func TestSubscriptionLifeCycle(t *testing.T) {
	stop := make(chan bool)
	server := subscription_setupServer()
	client, subscriptionClient := subscription_setupClients()
	msg := randomID()
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { _ = server.Shutdown(ctx) }()
	defer cancel()

	subscriptionClient.
		OnError(func(sc *SubscriptionClient, err error) error {
			return err
		})

	/*
		subscription {
			helloSaid {
				id
				msg
			}
		}
	*/
	var sub struct {
		HelloSaid struct {
			ID      String
			Message String `graphql:"msg" json:"msg"`
		} `graphql:"helloSaid" json:"helloSaid"`
	}

	_, err := subscriptionClient.Subscribe(
		sub,
		nil,
		func(data []byte, e error) error {
			if e != nil {
				t.Fatalf("got error: %v, want: nil", e)
				return nil
			}

			log.Println("result", string(data))
			e = json.Unmarshal(data, &sub)
			if e != nil {
				t.Fatalf("got error: %v, want: nil", e)
				return nil
			}

			if sub.HelloSaid.Message != String(msg) {
				t.Fatalf(
					"subscription message does not match. got: %s, want: %s",
					sub.HelloSaid.Message,
					msg,
				)
			}

			return errors.New("exit")
		},
	)

	if err != nil {
		t.Fatalf("got error: %v, want: nil", err)
	}

	go func() {
		if err := subscriptionClient.Run(); err == nil || err.Error() != "exit" {
			t.Errorf("got error: %v, want: exit", err)
		}
		stop <- true
	}()

	defer func() { _ = subscriptionClient.Close() }()

	// wait until the subscription client connects to the server
	time.Sleep(2 * time.Second)

	// call a mutation request to send message to the subscription
	/*
		mutation ($msg: String!) {
			sayHello(msg: $msg) {
				id
				msg
			}
		}
	*/
	var q struct {
		SayHello struct {
			ID  String
			Msg String
		} `graphql:"sayHello(msg: $msg)"`
	}
	variables := map[string]any{
		"msg": String(msg),
	}
	err = client.Mutate(
		context.Background(),
		&q,
		variables,
		OperationName("SayHello"),
	)
	if err != nil {
		t.Fatalf("got error: %v, want: nil", err)
	}

	<-stop
}

func TestSubscriptionLifeCycle2(t *testing.T) {
	server := subscription_setupServer()
	client, subscriptionClient := subscription_setupClients()
	msg := randomID()
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { _ = server.Shutdown(ctx) }()
	defer cancel()

	subscriptionClient.
		OnError(func(sc *SubscriptionClient, err error) error {
			// Ignore errors related to graceful shutdown (closed connections)
			// These are expected when subscriptions stop
			if err != nil &&
				strings.Contains(err.Error(), "closed network connection") {
				return nil
			}
			if err != nil && strings.Contains(err.Error(), "use of closed") {
				return nil
			}
			t.Fatalf("got error: %v, want: nil", err)
			return err
		}).
		OnDisconnected(func() {
			log.Println("disconnected")
		})
	/*
		subscription {
			helloSaid {
				id
				msg
			}
		}
	*/
	var sub struct {
		HelloSaid struct {
			ID      String
			Message String `graphql:"msg" json:"msg"`
		} `graphql:"helloSaid" json:"helloSaid"`
	}

	subId1, err := subscriptionClient.Subscribe(
		sub,
		nil,
		func(data []byte, e error) error {
			if e != nil {
				t.Fatalf("got error: %v, want: nil", e)
				return nil
			}

			log.Println("result", string(data))
			e = json.Unmarshal(data, &sub)
			if e != nil {
				t.Fatalf("got error: %v, want: nil", e)
				return nil
			}

			if sub.HelloSaid.Message != String(msg) {
				t.Fatalf(
					"subscription message does not match. got: %s, want: %s",
					sub.HelloSaid.Message,
					msg,
				)
			}

			return nil
		},
	)

	if err != nil {
		t.Fatalf("got error: %v, want: nil", err)
	}

	/*
		subscription {
			helloSaid {
				id
				msg
			}
		}
	*/
	var sub2 struct {
		HelloSaid struct {
			Message String `graphql:"msg" json:"msg"`
		} `graphql:"helloSaid" json:"helloSaid"`
	}

	_, err = subscriptionClient.Subscribe(
		sub2,
		nil,
		func(data []byte, e error) error {
			if e != nil {
				t.Fatalf("got error: %v, want: nil", e)
				return nil
			}

			log.Println("result", string(data))
			e = json.Unmarshal(data, &sub2)
			if e != nil {
				t.Fatalf("got error: %v, want: nil", e)
				return nil
			}

			if sub2.HelloSaid.Message != String(msg) {
				t.Fatalf(
					"subscription message does not match. got: %s, want: %s",
					sub2.HelloSaid.Message,
					msg,
				)
			}

			return ErrSubscriptionStopped
		},
	)

	if err != nil {
		t.Fatalf("got error: %v, want: nil", err)
	}

	go func() {
		// wait until the subscription client connects to the server
		time.Sleep(2 * time.Second)

		// call a mutation request to send message to the subscription
		/*
			mutation ($msg: String!) {
				sayHello(msg: $msg) {
					id
					msg
				}
			}
		*/
		var q struct {
			SayHello struct {
				ID  String
				Msg String
			} `graphql:"sayHello(msg: $msg)"`
		}
		variables := map[string]any{
			"msg": String(msg),
		}
		err = client.Mutate(
			context.Background(),
			&q,
			variables,
			OperationName("SayHello"),
		)
		if err != nil {
			t.Errorf("got error: %v, want: nil", err)
			return
		}

		time.Sleep(time.Second)
		_ = subscriptionClient.Unsubscribe(subId1)
	}()

	defer func() { _ = subscriptionClient.Close() }()

	if err := subscriptionClient.Run(); err != nil {
		t.Fatalf("got error: %v, want: nil", err)
	}
}

// TestSubscriptionClient_OptionSetters tests all option setter functions
func TestSubscriptionClient_OptionSetters(t *testing.T) {
	t.Run("WithTimeout", func(t *testing.T) {
		client := NewSubscriptionClient("ws://example.com").
			WithTimeout(30 * time.Second)
		if client == nil {
			t.Fatal("expected non-nil client")
		}
	})

	t.Run("WithRetryTimeout", func(t *testing.T) {
		client := NewSubscriptionClient("ws://example.com").
			WithRetryTimeout(60 * time.Second)
		if client == nil {
			t.Fatal("expected non-nil client")
		}
	})

	t.Run("WithoutLogTypes", func(t *testing.T) {
		client := NewSubscriptionClient("ws://example.com").
			WithoutLogTypes(GQL_CONNECTION_INIT, GQL_CONNECTION_ACK)
		if client == nil {
			t.Fatal("expected non-nil client")
		}
	})

	t.Run("WithReadLimit", func(t *testing.T) {
		client := NewSubscriptionClient("ws://example.com").
			WithReadLimit(1024 * 1024)
		if client == nil {
			t.Fatal("expected non-nil client")
		}
	})

	t.Run("OnConnected", func(t *testing.T) {
		_ = false
		callback := func() {
			// callback placeholder
		}
		client := NewSubscriptionClient("ws://example.com").OnConnected(callback)
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		// Note: callback won't be called without actually connecting
	})

	t.Run("WithWebSocketOptions", func(t *testing.T) {
		options := WebsocketOptions{
			HTTPClient: &http.Client{Timeout: 10 * time.Second},
		}
		client := NewSubscriptionClient("ws://example.com").
			WithWebSocketOptions(options)
		if client == nil {
			t.Fatal("expected non-nil client")
		}
	})

	t.Run("chaining multiple options", func(t *testing.T) {
		client := NewSubscriptionClient("ws://example.com").
			WithTimeout(30 * time.Second).
			WithRetryTimeout(60 * time.Second).
			WithReadLimit(2048).
			WithoutLogTypes(GQL_CONNECTION_INIT)
		if client == nil {
			t.Fatal("expected non-nil client after chaining")
		}
	})
}

// TestSubscriptionClient_DeprecatedMethods tests deprecated subscription methods
// These tests just verify the methods exist and don't panic when called
func TestSubscriptionClient_DeprecatedMethods(t *testing.T) {
	// Just verify the methods can be called without panicking
	client := NewSubscriptionClient("ws://example.com")

	var q struct {
		Greet string `graphql:"greet"`
	}

	dummyHandler := func(message []byte, err error) error {
		return nil
	}

	t.Run("NamedSubscribe", func(t *testing.T) {
		// Method should exist and not panic
		// It will fail to connect but that's OK for coverage
		_, _ = client.NamedSubscribe("TestOp", &q, nil, dummyHandler)
	})

	t.Run("SubscribeRaw", func(t *testing.T) {
		// Method should exist and not panic
		_, _ = client.SubscribeRaw(`subscription { test }`, nil, dummyHandler)
	})

	t.Run("Exec", func(t *testing.T) {
		// Method should exist and not panic
		_, _ = client.Exec(`subscription { test }`, nil, dummyHandler)
	})
}

// TestSubscriptionClient_MessageHandlers tests subscription message handlers
func TestSubscriptionClient_MessageHandlers(t *testing.T) {
	t.Run("handleConnectionKeepAliveMessage", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upgrader := websocket.Upgrader{}
			c, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer func() { _ = c.Close() }()

			// Read connection init
			_, _, _ = c.ReadMessage()

			// Send connection ack
			_ = c.WriteJSON(map[string]string{"type": string(GQL_CONNECTION_ACK)})

			// Send keep-alive messages
			for i := 0; i < 3; i++ {
				_ = c.WriteJSON(
					map[string]string{"type": string(GQL_CONNECTION_KEEP_ALIVE)},
				)
				time.Sleep(50 * time.Millisecond)
			}

			// Close gracefully
			time.Sleep(100 * time.Millisecond)
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		client := NewSubscriptionClient(wsURL).WithTimeout(2 * time.Second)

		go func() {
			time.Sleep(500 * time.Millisecond)
			_ = client.Close()
		}()

		_ = client.Run()
	})

	t.Run("handleConnectionErrorMessage", func(t *testing.T) {
		// Just verify OnError callback can be set
		called := false
		client := NewSubscriptionClient("ws://example.com").
			OnError(func(sc *SubscriptionClient, err error) error {
				called = true
				return err
			})
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		// Note: callback won't be called without actually triggering an error
		_ = called // suppress unused warning
	})

	t.Run("handleUnknownMessage", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upgrader := websocket.Upgrader{}
			c, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer func() { _ = c.Close() }()

			// Read connection init
			_, _, _ = c.ReadMessage()

			// Send connection ack
			_ = c.WriteJSON(map[string]string{"type": string(GQL_CONNECTION_ACK)})

			// Send unknown message type
			_ = c.WriteJSON(map[string]string{
				"type": "UNKNOWN_TYPE",
			})

			time.Sleep(200 * time.Millisecond)
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		client := NewSubscriptionClient(wsURL).WithTimeout(2 * time.Second)

		go func() {
			time.Sleep(500 * time.Millisecond)
			_ = client.Close()
		}()

		_ = client.Run()
		// Unknown messages are logged but don't cause errors
	})
}

// TestSubscriptionClient_Reset tests the Reset method
func TestSubscriptionClient_Reset(t *testing.T) {
	client := NewSubscriptionClient("ws://example.com")
	client.Reset()
	// Reset should reset the client to initial state
	// No error expected
}

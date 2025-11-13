package graphql_test

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/llehouerou/go-graphql-client"
)

func TestClient_Query_partialDataWithErrorResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, `{
			"data": {
				"node1": {
					"id": "MDEyOklzc3VlQ29tbWVudDE2OTQwNzk0Ng=="
				},
				"node2": null
			},
			"errors": [
				{
					"message": "Could not resolve to a node with the global id of 'NotExist'",
					"type": "NOT_FOUND",
					"path": [
						"node2"
					],
					"locations": [
						{
							"line": 10,
							"column": 4
						}
					]
				}
			]
		}`)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	var q struct {
		Node1 *struct {
			ID graphql.ID
		} `graphql:"node1: node(id: \"MDEyOklzc3VlQ29tbWVudDE2OTQwNzk0Ng==\")"`
		Node2 *struct {
			ID graphql.ID
		} `graphql:"node2: node(id: \"NotExist\")"`
	}

	_, err := client.QueryRaw(context.Background(), &q, nil)
	if err == nil {
		t.Fatal("got error: nil, want: non-nil")
	}

	err = client.Query(context.Background(), &q, nil)
	if err == nil {
		t.Fatal("got error: nil, want: non-nil")
	}
	if got, want := err.Error(), "Message: Could not resolve to a node with the global id of 'NotExist', Locations: [{Line:10 Column:4}]"; got != want {
		t.Errorf("got error: %v, want: %v", got, want)
	}

	if q.Node1 == nil || q.Node1.ID != "MDEyOklzc3VlQ29tbWVudDE2OTQwNzk0Ng==" {
		t.Errorf("got wrong q.Node1: %v", q.Node1)
	}
	if q.Node2 != nil {
		t.Errorf("got non-nil q.Node2: %v, want: nil", *q.Node2)
	}
}

func TestClient_Query_partialDataRawQueryWithErrorResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, `{
			"data": {
				"node1": { "id": "MDEyOklzc3VlQ29tbWVudDE2OTQwNzk0Ng==" },
				"node2": null
			},
			"errors": [
				{
					"message": "Could not resolve to a node with the global id of 'NotExist'",
					"type": "NOT_FOUND",
					"path": [
						"node2"
					],
					"locations": [
						{
							"line": 10,
							"column": 4
						}
					]
				}
			]
		}`)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	var q struct {
		Node1 json.RawMessage `graphql:"node1"`
		Node2 *struct {
			ID graphql.ID
		} `graphql:"node2: node(id: \"NotExist\")"`
	}
	err := client.Query(context.Background(), &q, nil)
	if err == nil {
		t.Fatal("got error: nil, want: non-nil\n")
	}
	if got, want := err.Error(), "Message: Could not resolve to a node with the global id of 'NotExist', Locations: [{Line:10 Column:4}]"; got != want {
		t.Errorf("got error: %v, want: %v\n", got, want)
	}
	if q.Node1 == nil ||
		string(q.Node1) != `{"id":"MDEyOklzc3VlQ29tbWVudDE2OTQwNzk0Ng=="}` {
		t.Errorf("got wrong q.Node1: %v\n", string(q.Node1))
	}
	if q.Node2 != nil {
		t.Errorf("got non-nil q.Node2: %v, want: nil\n", *q.Node2)
	}

	// test internal error data
	client = client.WithDebug(true)
	err = client.Query(context.Background(), &q, nil)
	if err == nil {
		t.Fatal("got error: nil, want: non-nil")
	}
	if !errors.As(err, &graphql.Errors{}) {
		t.Errorf("the error type should be graphql.Errors")
	}

	gqlErr := err.(graphql.Errors)
	if got, want := gqlErr[0].Message, `Could not resolve to a node with the global id of 'NotExist'`; got != want {
		t.Errorf("got error: %v, want: %v", got, want)
	}
}

func TestClient_Query_noDataWithErrorResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, `{
			"errors": [
				{
					"message": "Field 'user' is missing required arguments: login",
					"locations": [
						{
							"line": 7,
							"column": 3
						}
					]
				}
			]
		}`)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	var q struct {
		User struct {
			Name string
		}
	}
	err := client.Query(context.Background(), &q, nil)
	if err == nil {
		t.Fatal("got error: nil, want: non-nil")
	}
	if got, want := err.Error(), "Message: Field 'user' is missing required arguments: login, Locations: [{Line:7 Column:3}]"; got != want {
		t.Errorf("got error: %v, want: %v", got, want)
	}
	if q.User.Name != "" {
		t.Errorf("got non-empty q.User.Name: %v", q.User.Name)
	}

	_, err = client.QueryRaw(context.Background(), &q, nil)
	if err == nil {
		t.Fatal("got error: nil, want: non-nil")
	}

	// test internal error data
	client = client.WithDebug(true)
	err = client.Query(context.Background(), &q, nil)
	if err == nil {
		t.Fatal("got error: nil, want: non-nil")
	}
	if !errors.As(err, &graphql.Errors{}) {
		t.Errorf("the error type should be graphql.Errors")
	}

	gqlErr := err.(graphql.Errors)
	if got, want := gqlErr[0].Message, `Field 'user' is missing required arguments: login`; got != want {
		t.Errorf("got error: %v, want: %v", got, want)
	}

	interErr := gqlErr[0].Extensions["internal"].(map[string]any)

	if got, want := interErr["request"].(map[string]any)["body"], "{\"query\":\"{user{name}}\"}\n"; got != want {
		t.Errorf("got error: %v, want: %v", got, want)
	}
}

func TestClient_Query_errorStatusCode(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "important message", http.StatusInternalServerError)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	var q struct {
		User struct {
			Name string
		}
	}
	err := client.Query(context.Background(), &q, nil)
	if err == nil {
		t.Fatal("got error: nil, want: non-nil")
	}
	if got, want := err.Error(), `Message: 500 Internal Server Error; body: "important message\n", Locations: []`; got != want {
		t.Errorf("got error: %v, want: %v", got, want)
	}
	if q.User.Name != "" {
		t.Errorf("got non-empty q.User.Name: %v", q.User.Name)
	}

	gqlErr := err.(graphql.Errors)
	if got, want := gqlErr[0].Extensions["code"], graphql.ErrRequestError; got != want {
		t.Errorf("got error: %v, want: %v", got, want)
	}
	if _, ok := gqlErr[0].Extensions["internal"]; ok {
		t.Errorf("expected empty internal error")
	}

	// test internal error data
	client = client.WithDebug(true)
	err = client.Query(context.Background(), &q, nil)
	if err == nil {
		t.Fatal("got error: nil, want: non-nil")
	}
	if !errors.As(err, &graphql.Errors{}) {
		t.Errorf("the error type should be graphql.Errors")
	}
	gqlErr = err.(graphql.Errors)
	if got, want := gqlErr[0].Message, `500 Internal Server Error; body: "important message\n"`; got != want {
		t.Errorf("got error: %v, want: %v", got, want)
	}
	if got, want := gqlErr[0].Extensions["code"], graphql.ErrRequestError; got != want {
		t.Errorf("got error: %v, want: %v", got, want)
	}
	interErr := gqlErr[0].Extensions["internal"].(map[string]any)

	if got, want := interErr["request"].(map[string]any)["body"], "{\"query\":\"{user{name}}\"}\n"; got != want {
		t.Errorf("got error: %v, want: %v", got, want)
	}
}

// Test that an empty (but non-nil) variables map is
// handled no differently than a nil variables map.
func TestClient_Query_emptyVariables(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		body := mustRead(req.Body)
		if got, want := body, `{"query":"{user{name}}"}`+"\n"; got != want {
			t.Errorf("got body: %v, want %v", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, `{"data": {"user": {"name": "Gopher"}}}`)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	var q struct {
		User struct {
			Name string
		}
	}
	err := client.Query(context.Background(), &q, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := q.User.Name, "Gopher"; got != want {
		t.Errorf("got q.User.Name: %q, want: %q", got, want)
	}
}

// Test ignored field
// handled no differently than a nil variables map.
func TestClient_Query_ignoreFields(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		body := mustRead(req.Body)
		if got, want := body, `{"query":"{user{id,name}}"}`+"\n"; got != want {
			t.Errorf("got body: %v, want %v", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, `{"data": {"user": {"name": "Gopher"}}}`)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	var q struct {
		User struct {
			ID      string `graphql:"id"`
			Name    string `graphql:"name"`
			Ignored string `graphql:"-"`
		}
	}
	err := client.Query(context.Background(), &q, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := q.User.Name, "Gopher"; got != want {
		t.Errorf("got q.User.Name: %q, want: %q", got, want)
	}
	if got, want := q.User.Ignored, ""; got != want {
		t.Errorf("got q.User.Ignored: %q, want: %q", got, want)
	}
}

// Test raw json response from query
func TestClient_Query_RawResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		body := mustRead(req.Body)
		if got, want := body, `{"query":"{user{id,name}}"}`+"\n"; got != want {
			t.Errorf("got body: %v, want %v", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, `{"data": {"user": {"name": "Gopher"}}}`)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	var q struct {
		User struct {
			ID   string `graphql:"id"`
			Name string `graphql:"name"`
		}
	}
	rawBytes, err := client.QueryRaw(context.Background(), &q, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(rawBytes, &q)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := q.User.Name, "Gopher"; got != want {
		t.Errorf("got q.User.Name: %q, want: %q", got, want)
	}
}

// Test exec pre-built query
func TestClient_Exec_Query(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		body := mustRead(req.Body)
		if got, want := body, `{"query":"{user{id,name}}"}`+"\n"; got != want {
			t.Errorf("got body: %v, want %v", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, `{"data": {"user": {"name": "Gopher"}}}`)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	var q struct {
		User struct {
			ID   string `graphql:"id"`
			Name string `graphql:"name"`
		}
	}

	err := client.Exec(
		context.Background(),
		"{user{id,name}}",
		&q,
		map[string]any{},
	)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := q.User.Name, "Gopher"; got != want {
		t.Errorf("got q.User.Name: %q, want: %q", got, want)
	}
}

// Test exec pre-built query, return raw json string
func TestClient_Exec_QueryRaw(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		body := mustRead(req.Body)
		if got, want := body, `{"query":"{user{id,name}}"}`+"\n"; got != want {
			t.Errorf("got body: %v, want %v", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, `{"data": {"user": {"name": "Gopher"}}}`)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	var q struct {
		User struct {
			ID   string `graphql:"id"`
			Name string `graphql:"name"`
		}
	}

	rawBytes, err := client.ExecRaw(
		context.Background(),
		"{user{id,name}}",
		map[string]any{},
	)
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(rawBytes, &q)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := q.User.Name, "Gopher"; got != want {
		t.Errorf("got q.User.Name: %q, want: %q", got, want)
	}
}

// localRoundTripper is an http.RoundTripper that executes HTTP transactions
// by using handler directly, instead of going over an HTTP connection.
type localRoundTripper struct {
	handler http.Handler
}

func (l localRoundTripper) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	w := httptest.NewRecorder()
	l.handler.ServeHTTP(w, req)
	return w.Result(), nil
}

func mustRead(r io.Reader) string {
	b, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func mustWrite(w io.Writer, s string) {
	_, err := io.WriteString(w, s)
	if err != nil {
		panic(err)
	}
}

type Id struct {
	Type string
	ID   string
}

type Wrapped struct {
	Value1 string `graphql:"value1"`
	Value2 Id     `graphql:"value2"`
}

type Wrapper[T any] struct {
	Value T
}

func (w Wrapper[T]) GetGraphQLType() string {
	return "wrapper"
}

func (w Wrapper[T]) GetGraphQLWrapped() T {
	return w.Value
}

func (w Wrapper[T]) GetInnerLayer() ContainerLayer[T] {
	return nil
}

type ActualNodes[T any] struct {
	gqlType string `graphql:"-"`
	Nodes   T
}

func (an *ActualNodes[T]) GetInnerLayer() ContainerLayer[T] {
	return nil
}

func (an *ActualNodes[T]) GetNodes() T {
	return an.Nodes
}

func (an *ActualNodes[T]) GetGraphQLType() string {
	return an.gqlType
}

type ContainerLayer[T any] interface {
	GetInnerLayer() ContainerLayer[T]
	GetNodes() T
	GetGraphQLType() string
}

type NestedLayer[T any] struct {
	gqlType    string `graphql:"-"`
	InnerLayer ContainerLayer[T]
}

func (nl *NestedLayer[T]) GetInnerLayer() ContainerLayer[T] {
	return nl.InnerLayer
}

func (nl *NestedLayer[T]) GetNodes() T {
	var res T
	return res
}

func (nl *NestedLayer[T]) GetGraphQLType() string {
	return nl.gqlType
}

type NestedQuery[T any] struct {
	OutermostLayer ContainerLayer[T]
}

func (q *NestedQuery[T]) GetNodes() T {
	if q.OutermostLayer == nil {
		var res T
		return res
	}
	layer := q.OutermostLayer
	for layer.GetInnerLayer() != nil {
		layer = layer.GetInnerLayer()
	}
	return layer.GetNodes()
}

func NewNestedQuery[T any](containerLayers ...string) *NestedQuery[T] {
	if len(containerLayers) == 0 {
		return &NestedQuery[T]{
			OutermostLayer: &ActualNodes[T]{},
		}
	}

	var buildLayer func(index int) ContainerLayer[T]
	buildLayer = func(index int) ContainerLayer[T] {
		if index == len(containerLayers)-1 {
			return &ActualNodes[T]{
				gqlType: containerLayers[index],
			}
		}
		return &NestedLayer[T]{
			gqlType:    containerLayers[index],
			InnerLayer: buildLayer(index + 1),
		}
	}

	return &NestedQuery[T]{OutermostLayer: buildLayer(0)}
}

func TestClient_Query_withWrapper(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		body := mustRead(req.Body)
		if got, want := body, `{"query":"{testcontainer{wrapper{value1,value2{type,id}}}}"}`+"\n"; got != want {
			t.Errorf("got body: %v, want %v", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		mustWrite(
			w,
			`{"data": {"testcontainer": { "wrapper": {"value1": "Gopher", "value2": {"type": "test", "id": "123"}}}}}}`,
		)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	q := NewNestedQuery[Wrapper[Wrapped]]("testcontainer")
	err := client.Query(context.Background(), &q, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := q.GetNodes().Value.Value1, "Gopher"; got != want {
		t.Errorf("got q.User.Name: %q, want: %q", got, want)
	}
	if got, want := q.GetNodes().Value.Value2.Type, "test"; got != want {
		t.Errorf("got q.User.Name: %q, want: %q", got, want)
	}
	if got, want := q.GetNodes().Value.Value2.ID, "123"; got != want {
		t.Errorf("got q.User.Name: %q, want: %q", got, want)
	}

}

// TestClient_Query_multiLevelNesting tests wrapper with multiple nesting levels
// to validate the GetNodes() traversal logic.
func TestClient_Query_multiLevelNesting(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		body := mustRead(req.Body)
		expected := `{"query":"{layer1{layer2{layer3{wrapper{value1,value2{type,id}}}}}}"}`+ "\n"
		if got := body; got != expected {
			t.Errorf("got body: %v, want %v", got, expected)
		}
		w.Header().Set("Content-Type", "application/json")
		mustWrite(
			w,
			`{"data": {"layer1": {"layer2": {"layer3": {"wrapper": {"value1": "Deep", "value2": {"type": "nested", "id": "456"}}}}}}}`,
		)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	// Create nested query with 3 container layers
	q := NewNestedQuery[Wrapper[Wrapped]]("layer1", "layer2", "layer3")
	err := client.Query(context.Background(), &q, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	// Verify GetNodes() correctly traverses all layers
	nodes := q.GetNodes()
	if got, want := nodes.Value.Value1, "Deep"; got != want {
		t.Errorf("got Value1: %q, want: %q", got, want)
	}
	if got, want := nodes.Value.Value2.Type, "nested"; got != want {
		t.Errorf("got Type: %q, want: %q", got, want)
	}
	if got, want := nodes.Value.Value2.ID, "456"; got != want {
		t.Errorf("got ID: %q, want: %q", got, want)
	}
}

// TestClient_Mutation_withWrapper tests mutations with wrapped types
// to ensure wrappers work correctly in mutation operations.
func TestClient_Mutation_withWrapper(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		body := mustRead(req.Body)
		// Note: Mutation with variables includes type definition
		expected := `{"query":"mutation ($name:String!){createUser(name: $name){wrapper{value1,value2{type,id}}}}","variables":{"name":"Alice"}}`+ "\n"
		if got := body; got != expected {
			t.Errorf("got body: %v, want %v", got, expected)
		}
		w.Header().Set("Content-Type", "application/json")
		mustWrite(
			w,
			`{"data": {"createUser": {"wrapper": {"value1": "Alice", "value2": {"type": "user", "id": "789"}}}}}`,
		)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	var m struct {
		CreateUser struct {
			Wrapper Wrapper[Wrapped]
		} `graphql:"createUser(name: $name)"`
	}

	variables := map[string]any{
		"name": "Alice",
	}

	err := client.Mutate(context.Background(), &m, variables)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := m.CreateUser.Wrapper.Value.Value1, "Alice"; got != want {
		t.Errorf("got Value1: %q, want: %q", got, want)
	}
	if got, want := m.CreateUser.Wrapper.Value.Value2.Type, "user"; got != want {
		t.Errorf("got Type: %q, want: %q", got, want)
	}
	if got, want := m.CreateUser.Wrapper.Value.Value2.ID, "789"; got != want {
		t.Errorf("got ID: %q, want: %q", got, want)
	}
}

// TestClient_Query_StructVariables tests end-to-end client Query with struct variables
// This validates the struct-based variable support added in commit e2d1096.
func TestClient_Query_StructVariables(t *testing.T) {
	tests := []struct {
		name          string
		variables     any
		responseBody  string
		validateQuery func(t *testing.T, q any)
		validateVars  func(t *testing.T, vars map[string]any)
	}{
		{
			name: "struct with basic types",
			variables: struct {
				CharacterID graphql.ID `json:"characterId"`
				Name        string     `json:"name"`
			}{
				CharacterID: graphql.ID("1003"),
				Name:        "Han Solo",
			},
			responseBody: `{"data":{"hero":{"name":"Han Solo"}}}`,
			validateQuery: func(t *testing.T, q any) {
				query := q.(*struct {
					Hero struct {
						Name string
					} `graphql:"hero(id: $characterId, name: $name)"`
				})
				if got, want := query.Hero.Name, "Han Solo"; got != want {
					t.Errorf("got Hero.Name: %q, want: %q", got, want)
				}
			},
			validateVars: func(t *testing.T, vars map[string]any) {
				if got, want := vars["characterId"], "1003"; got != want {
					t.Errorf(
						"got characterId: %v, want: %v",
						got,
						want,
					)
				}
				if got, want := vars["name"], "Han Solo"; got != want {
					t.Errorf("got name: %v, want: %v", got, want)
				}
			},
		},
		{
			name: "struct with pointer fields (nullable)",
			variables: struct {
				CharacterID *graphql.ID `json:"characterId"`
				Name        *string     `json:"name"`
			}{
				CharacterID: graphql.NewID("1003"),
				Name:        stringPtr("Luke"),
			},
			responseBody: `{"data":{"hero":{"name":"Luke Skywalker"}}}`,
			validateQuery: func(t *testing.T, q any) {
				query := q.(*struct {
					Hero struct {
						Name string
					} `graphql:"hero(id: $characterId, name: $name)"`
				})
				if got, want := query.Hero.Name, "Luke Skywalker"; got != want {
					t.Errorf("got Hero.Name: %q, want: %q", got, want)
				}
			},
			validateVars: func(t *testing.T, vars map[string]any) {
				if got, want := vars["characterId"], "1003"; got != want {
					t.Errorf("got characterId: %v, want: %v", got, want)
				}
				if got, want := vars["name"], "Luke"; got != want {
					t.Errorf("got name: %v, want: %v", got, want)
				}
			},
		},
		{
			name: "backward compatibility with map",
			variables: map[string]any{
				"characterId": graphql.ID("2000"),
			},
			responseBody: `{"data":{"hero":{"name":"C-3PO"}}}`,
			validateQuery: func(t *testing.T, q any) {
				query := q.(*struct {
					Hero struct {
						Name string
					} `graphql:"hero(id: $characterId)"`
				})
				if got, want := query.Hero.Name, "C-3PO"; got != want {
					t.Errorf("got Hero.Name: %q, want: %q", got, want)
				}
			},
			validateVars: func(t *testing.T, vars map[string]any) {
				// JSON unmarshaling converts graphql.ID to string
				if got, want := vars["characterId"], "2000"; got != want {
					t.Errorf("got characterId: %v, want: %v", got, want)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
				body := mustRead(req.Body)

				// Parse the request to validate variables were properly serialized
				var reqBody struct {
					Query     string         `json:"query"`
					Variables map[string]any `json:"variables,omitempty"`
				}
				if err := json.Unmarshal([]byte(body), &reqBody); err != nil {
					t.Fatalf(
						"failed to unmarshal request body: %v",
						err,
					)
				}

				// Validate variables if test specifies validation
				if tc.validateVars != nil && len(reqBody.Variables) > 0 {
					tc.validateVars(t, reqBody.Variables)
				}

				w.Header().Set("Content-Type", "application/json")
				mustWrite(w, tc.responseBody)
			})
			client := graphql.NewClient(
				"/graphql",
				&http.Client{Transport: localRoundTripper{handler: mux}},
			)

			// Build the appropriate query struct based on test case
			var q any
			switch tc.name {
			case "struct with basic types", "struct with pointer fields (nullable)":
				q = &struct {
					Hero struct {
						Name string
					} `graphql:"hero(id: $characterId, name: $name)"`
				}{}
			case "backward compatibility with map":
				q = &struct {
					Hero struct {
						Name string
					} `graphql:"hero(id: $characterId)"`
				}{}
			}

			err := client.Query(context.Background(), q, tc.variables)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.validateQuery != nil {
				tc.validateQuery(t, q)
			}
		})
	}
}

// TestClient_Mutate_StructVariables tests end-to-end client Mutate with struct variables
func TestClient_Mutate_StructVariables(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		body := mustRead(req.Body)

		// Parse and validate variables were serialized
		var reqBody struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.Unmarshal([]byte(body), &reqBody); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}

		// Validate struct variables were properly serialized
		if got, want := reqBody.Variables["userId"], "456"; got != want {
			t.Errorf("got userId: %v, want: %v", got, want)
		}
		if got, want := reqBody.Variables["name"], "Jane Smith"; got != want {
			t.Errorf("got name: %v, want: %v", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, `{"data":{"updateUser":{"id":"456","name":"Jane Smith"}}}`)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	variables := struct {
		UserID graphql.ID `json:"userId"`
		Name   string     `json:"name"`
	}{
		UserID: graphql.ID("456"),
		Name:   "Jane Smith",
	}

	var m struct {
		UpdateUser struct {
			ID   graphql.ID
			Name string
		} `graphql:"updateUser(id: $userId, name: $name)"`
	}

	err := client.Mutate(context.Background(), &m, variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := m.UpdateUser.ID, graphql.ID("456"); got != want {
		t.Errorf("got UpdateUser.ID: %v, want: %v", got, want)
	}
	if got, want := m.UpdateUser.Name, "Jane Smith"; got != want {
		t.Errorf("got UpdateUser.Name: %v, want: %v", got, want)
	}
}

// TestClient_QueryRaw_StructVariables tests QueryRaw with struct variables
// Validates that struct variables are properly serialized in HTTP request
func TestClient_QueryRaw_StructVariables(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		body := mustRead(req.Body)

		// Verify the variables are properly serialized
		var reqBody struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.Unmarshal([]byte(body), &reqBody); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}

		// The key validation: struct variables were serialized to JSON
		if got, want := reqBody.Variables["characterId"], "1003"; got != want {
			t.Errorf("got characterId: %q, want: %q", got, want)
		}
		if got, want := reqBody.Variables["name"], "Han Solo"; got != want {
			t.Errorf("got name: %q, want: %q", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, `{"data":{"hero":{"name":"Han Solo"}}}`)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	variables := struct {
		CharacterID graphql.ID `json:"characterId"`
		Name        string     `json:"name"`
	}{
		CharacterID: graphql.ID("1003"),
		Name:        "Han Solo",
	}

	var q struct {
		Hero struct {
			Name string
		} `graphql:"hero(id: $characterId, name: $name)"`
	}

	// QueryRaw returns raw bytes and populates the struct
	rawResp, err := client.QueryRaw(context.Background(), &q, variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify we got a response
	if len(rawResp) == 0 {
		t.Error("expected non-empty raw response")
	}
}

func stringPtr(s string) *string {
	return &s
}

// TestClient_decorateError tests the helper method for decorating errors
// based on debug mode settings.
func TestClient_decorateError(t *testing.T) {
	t.Run("debug mode enabled with request and response", func(t *testing.T) {
		client := graphql.NewClient("http://example.com", nil).WithDebug(true)

		// Create a mock request and response
		reqBody := `{"query":"{test}"}`
		req, err := http.NewRequest(http.MethodPost, "http://example.com", nil)
		if err != nil {
			t.Fatal(err)
		}

		resp := &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}

		baseErr := graphql.Error{
			Message: "test error",
			Extensions: map[string]any{
				"code": graphql.ErrRequestError,
			},
		}

		// Call the helper method (which doesn't exist yet - this should fail)
		decorated := client.DecorateError(
			baseErr,
			req,
			resp,
			strings.NewReader(reqBody),
			strings.NewReader(`{"data":null}`),
		)

		// Verify request information was added
		if decorated.Extensions == nil {
			t.Fatal("expected Extensions to be non-nil")
		}

		internal, ok := decorated.Extensions["internal"].(map[string]any)
		if !ok {
			t.Fatal("expected internal extensions to exist")
		}

		if _, ok := internal["request"]; !ok {
			t.Error("expected request information in internal extensions")
		}

		if _, ok := internal["response"]; !ok {
			t.Error("expected response information in internal extensions")
		}
	})

	t.Run("debug mode disabled", func(t *testing.T) {
		client := graphql.NewClient("http://example.com", nil).WithDebug(false)

		reqBody := `{"query":"{test}"}`
		req, err := http.NewRequest(http.MethodPost, "http://example.com", nil)
		if err != nil {
			t.Fatal(err)
		}

		resp := &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}

		baseErr := graphql.Error{
			Message: "test error",
			Extensions: map[string]any{
				"code": graphql.ErrRequestError,
			},
		}

		decorated := client.DecorateError(
			baseErr,
			req,
			resp,
			strings.NewReader(reqBody),
			strings.NewReader(`{"data":null}`),
		)

		// In non-debug mode, internal extensions should not be added
		if decorated.Extensions != nil {
			if internal, ok := decorated.Extensions["internal"].(map[string]any); ok {
				if len(internal) > 0 {
					t.Error("expected no internal extensions in non-debug mode")
				}
			}
		}

		// The base error should still have the code
		if code, ok := decorated.Extensions["code"].(string); !ok || code != graphql.ErrRequestError {
			t.Error("expected code to be preserved")
		}
	})
}

// TestClient_newRequestError tests the convenience method for creating
// and decorating errors in one step.
func TestClient_newRequestError(t *testing.T) {
	t.Run("creates error with code and message", func(t *testing.T) {
		client := graphql.NewClient("http://example.com", nil)

		err := client.NewRequestError(
			graphql.ErrJsonDecode,
			errors.New("json decode failed"),
			nil,
			nil,
			nil,
			nil,
		)

		if err.Message != "json decode failed" {
			t.Errorf("expected message 'json decode failed', got %q", err.Message)
		}

		if code, ok := err.Extensions["code"].(string); !ok || code != graphql.ErrJsonDecode {
			t.Errorf("expected code %q, got %v", graphql.ErrJsonDecode, code)
		}
	})

	t.Run("decorates with debug info when enabled", func(t *testing.T) {
		client := graphql.NewClient("http://example.com", nil).WithDebug(true)

		reqBody := `{"query":"{test}"}`
		req, err := http.NewRequest(http.MethodPost, "http://example.com", nil)
		if err != nil {
			t.Fatal(err)
		}

		resp := &http.Response{
			StatusCode: 500,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}

		decoratedErr := client.NewRequestError(
			graphql.ErrRequestError,
			errors.New("server error"),
			req,
			resp,
			strings.NewReader(reqBody),
			strings.NewReader(`{"errors":[]}`),
		)

		if decoratedErr.Message != "server error" {
			t.Errorf("expected message 'server error', got %q", decoratedErr.Message)
		}

		internal, ok := decoratedErr.Extensions["internal"].(map[string]any)
		if !ok {
			t.Fatal("expected internal extensions in debug mode")
		}

		if _, ok := internal["request"]; !ok {
			t.Error("expected request information in debug mode")
		}

		if _, ok := internal["response"]; !ok {
			t.Error("expected response information in debug mode")
		}
	})
}

// TestClient_buildRequest tests the buildRequest method that constructs
// the HTTP request with JSON body for GraphQL operations
func TestClient_buildRequest(t *testing.T) {
	t.Run("builds request with query and variables", func(t *testing.T) {
		client := graphql.NewClient("http://example.com/graphql", nil)
		ctx := context.Background()
		query := "{user{name}}"
		variables := map[string]any{"id": "123"}

		req, reqBody, err := client.BuildRequest(ctx, query, variables)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if req.Method != http.MethodPost {
			t.Errorf("expected method POST, got %s", req.Method)
		}

		if req.URL.String() != "http://example.com/graphql" {
			t.Errorf("expected URL http://example.com/graphql, got %s", req.URL.String())
		}

		if contentType := req.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}

		var body struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables,omitempty"`
		}
		if err := json.Unmarshal(reqBody, &body); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}

		if body.Query != query {
			t.Errorf("expected query %q, got %q", query, body.Query)
		}

		if body.Variables["id"] != "123" {
			t.Errorf("expected variables[id]=123, got %v", body.Variables["id"])
		}
	})

	t.Run("builds request without variables", func(t *testing.T) {
		client := graphql.NewClient("http://example.com/graphql", nil)
		ctx := context.Background()
		query := "{user{name}}"

		req, reqBody, err := client.BuildRequest(ctx, query, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if req == nil {
			t.Fatal("expected non-nil request")
		}

		var body struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables,omitempty"`
		}
		if err := json.Unmarshal(reqBody, &body); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}

		if body.Query != query {
			t.Errorf("expected query %q, got %q", query, body.Query)
		}

		if body.Variables != nil {
			t.Errorf("expected nil variables, got %v", body.Variables)
		}
	})

	t.Run("applies request modifier", func(t *testing.T) {
		modifierCalled := false
		client := graphql.NewClient("http://example.com/graphql", nil).
			WithRequestModifier(func(req *http.Request) {
				modifierCalled = true
				req.Header.Set("Authorization", "Bearer token123")
			})

		ctx := context.Background()
		query := "{user{name}}"

		req, _, err := client.BuildRequest(ctx, query, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !modifierCalled {
			t.Error("expected request modifier to be called")
		}

		if auth := req.Header.Get("Authorization"); auth != "Bearer token123" {
			t.Errorf("expected Authorization header 'Bearer token123', got %q", auth)
		}
	})
}

// TestClient_ImmutablePattern tests that With* methods follow the immutable
// pattern by returning new Client instances without modifying the original.
func TestClient_ImmutablePattern(t *testing.T) {
	t.Run("WithDebug returns new instance", func(t *testing.T) {
		original := graphql.NewClient("http://example.com/graphql", nil)

		// Call WithDebug and verify it returns a different instance
		modified := original.WithDebug(true)

		// The returned client should be a different instance
		if modified == original {
			t.Error("WithDebug returned the same instance (expected new instance)")
		}

		// Further modification should not affect the first modified instance
		modified2 := modified.WithDebug(false)
		if modified2 == modified || modified2 == original {
			t.Error("second WithDebug call should return yet another new instance")
		}
	})

	t.Run("WithRequestModifier returns new instance", func(t *testing.T) {
		original := graphql.NewClient("http://example.com/graphql", nil)

		// Original has no modifier
		ctx := context.Background()
		req, _, err := original.BuildRequest(ctx, "{test}", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.Header.Get("X-Test") != "" {
			t.Error("original client should not have the header")
		}

		// Call WithRequestModifier but don't capture the result
		_ = original.WithRequestModifier(func(r *http.Request) {
			r.Header.Set("X-Test", "modified")
		})

		// Original should still have no modifier effect
		req, _, err = original.BuildRequest(ctx, "{test}", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.Header.Get("X-Test") != "" {
			t.Error("WithRequestModifier modified the original client (expected immutable)")
		}

		// Captured result should have the modifier
		modified := original.WithRequestModifier(func(r *http.Request) {
			r.Header.Set("X-Test", "modified")
		})
		req, _, err = modified.BuildRequest(ctx, "{test}", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.Header.Get("X-Test") != "modified" {
			t.Error("WithRequestModifier didn't return a client with the modifier")
		}
	})

	t.Run("chaining With methods works correctly", func(t *testing.T) {
		original := graphql.NewClient("http://example.com/graphql", nil)

		// Chain methods
		modified := original.
			WithDebug(true).
			WithRequestModifier(func(r *http.Request) {
				r.Header.Set("X-Chain", "test")
			})

		// Modified client should be a different instance
		if modified == original {
			t.Error("chained client should be a new instance")
		}

		// Modified client should have the modifier
		ctx := context.Background()
		req, _, err := modified.BuildRequest(ctx, "{test}", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.Header.Get("X-Chain") != "test" {
			t.Error("chained client should have the modifier")
		}

		// Original should not have the modifier
		req, _, err = original.BuildRequest(ctx, "{test}", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.Header.Get("X-Chain") != "" {
			t.Error("original client should not have the modifier")
		}
	})

	t.Run("chaining order doesn't matter", func(t *testing.T) {
		original := graphql.NewClient("http://example.com/graphql", nil)

		// Chain in one order
		client1 := original.
			WithDebug(true).
			WithRequestModifier(func(r *http.Request) {
				r.Header.Set("X-Order", "1")
			})

		// Chain in reverse order
		client2 := original.
			WithRequestModifier(func(r *http.Request) {
				r.Header.Set("X-Order", "2")
			}).
			WithDebug(true)

		// Both should be new instances
		if client1 == original || client2 == original || client1 == client2 {
			t.Error("all clients should be different instances")
		}

		// Both should have their modifiers
		ctx := context.Background()
		req1, _, err := client1.BuildRequest(ctx, "{test}", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req1.Header.Get("X-Order") != "1" {
			t.Error("client1 should have its modifier")
		}

		req2, _, err := client2.BuildRequest(ctx, "{test}", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req2.Header.Get("X-Order") != "2" {
			t.Error("client2 should have its modifier")
		}
	})
}

// TestClient_executeRequest tests the executeRequest method that executes
// the HTTP request and handles gzip decompression
func TestClient_executeRequest(t *testing.T) {
	t.Run("executes request successfully", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			mustWrite(w, `{"data":{"user":{"name":"Alice"}}}`)
		})

		client := graphql.NewClient(
			"/graphql",
			&http.Client{Transport: localRoundTripper{handler: mux}},
		)

		req, err := http.NewRequest(http.MethodPost, "/graphql", strings.NewReader("{}"))
		if err != nil {
			t.Fatal(err)
		}

		resp, reader, execErr := client.ExecuteRequest(req)
		if execErr != nil {
			t.Fatalf("unexpected error: %v", execErr)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("failed to read response: %v", err)
		}

		expected := `{"data":{"user":{"name":"Alice"}}}`
		if string(body) != expected {
			t.Errorf("expected body %q, got %q", expected, string(body))
		}
	})

	t.Run("handles non-200 status code", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		})

		client := graphql.NewClient(
			"/graphql",
			&http.Client{Transport: localRoundTripper{handler: mux}},
		)

		req, err := http.NewRequest(http.MethodPost, "/graphql", strings.NewReader("{}"))
		if err != nil {
			t.Fatal(err)
		}

		_, _, execErr := client.ExecuteRequest(req)
		if execErr == nil {
			t.Fatal("expected error for non-200 status, got nil")
		}

		if !strings.Contains(execErr.Error(), "500") {
			t.Errorf("expected error to mention 500 status, got %q", execErr.Error())
		}
	})

	t.Run("handles gzip compression", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip")

			gzWriter := gzip.NewWriter(w)
			defer func() { _ = gzWriter.Close() }()
			_, _ = gzWriter.Write([]byte(`{"data":{"user":{"name":"Bob"}}}`))
		})

		client := graphql.NewClient(
			"/graphql",
			&http.Client{Transport: localRoundTripper{handler: mux}},
		)

		req, err := http.NewRequest(http.MethodPost, "/graphql", strings.NewReader("{}"))
		if err != nil {
			t.Fatal(err)
		}

		resp, reader, execErr := client.ExecuteRequest(req)
		if execErr != nil {
			t.Fatalf("unexpected error: %v", execErr)
		}

		if resp.Header.Get("Content-Encoding") != "gzip" {
			t.Errorf("expected Content-Encoding gzip, got %q", resp.Header.Get("Content-Encoding"))
		}

		body, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("failed to read response: %v", err)
		}

		expected := `{"data":{"user":{"name":"Bob"}}}`
		if string(body) != expected {
			t.Errorf("expected body %q, got %q", expected, string(body))
		}
	})
}

// TestClient_decodeResponse tests the decodeResponse method that decodes
// GraphQL JSON responses into data and errors
func TestClient_decodeResponse(t *testing.T) {
	t.Run("decodes successful response with data", func(t *testing.T) {
		client := graphql.NewClient("http://example.com/graphql", nil)
		responseBody := `{"data":{"user":{"name":"Alice","id":"123"}}}`
		reader := strings.NewReader(responseBody)

		rawData, errs := client.DecodeResponse(reader)
		if errs != nil {
			t.Fatalf("unexpected errors: %v", errs)
		}

		var result struct {
			User struct {
				Name string `json:"name"`
				ID   string `json:"id"`
			} `json:"user"`
		}
		if err := json.Unmarshal(rawData, &result); err != nil {
			t.Fatalf("failed to unmarshal raw data: %v", err)
		}

		if result.User.Name != "Alice" {
			t.Errorf("expected name Alice, got %s", result.User.Name)
		}
		if result.User.ID != "123" {
			t.Errorf("expected id 123, got %s", result.User.ID)
		}
	})

	t.Run("decodes response with errors", func(t *testing.T) {
		client := graphql.NewClient("http://example.com/graphql", nil)
		responseBody := `{"errors":[{"message":"field not found","locations":[{"line":1,"column":2}]}]}`
		reader := strings.NewReader(responseBody)

		rawData, errs := client.DecodeResponse(reader)
		if errs == nil {
			t.Fatal("expected errors, got nil")
		}

		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d", len(errs))
		}

		if errs[0].Message != "field not found" {
			t.Errorf("expected message 'field not found', got %q", errs[0].Message)
		}

		if rawData != nil {
			t.Errorf("expected nil raw data with errors only, got %s", string(rawData))
		}
	})

	t.Run("decodes response with partial data and errors", func(t *testing.T) {
		client := graphql.NewClient("http://example.com/graphql", nil)
		responseBody := `{"data":{"user":{"name":"Bob"}},"errors":[{"message":"some field failed"}]}`
		reader := strings.NewReader(responseBody)

		rawData, errs := client.DecodeResponse(reader)
		if errs == nil {
			t.Fatal("expected errors, got nil")
		}

		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d", len(errs))
		}

		if errs[0].Message != "some field failed" {
			t.Errorf("expected message 'some field failed', got %q", errs[0].Message)
		}

		// Should still have partial data
		if rawData == nil {
			t.Fatal("expected raw data with partial response, got nil")
		}

		var result struct {
			User struct {
				Name string `json:"name"`
			} `json:"user"`
		}
		if err := json.Unmarshal(rawData, &result); err != nil {
			t.Fatalf("failed to unmarshal partial data: %v", err)
		}

		if result.User.Name != "Bob" {
			t.Errorf("expected name Bob, got %s", result.User.Name)
		}
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		client := graphql.NewClient("http://example.com/graphql", nil)
		responseBody := `{invalid json}`
		reader := strings.NewReader(responseBody)

		_, errs := client.DecodeResponse(reader)
		if errs == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}

		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d", len(errs))
		}

		if code, ok := errs[0].Extensions["code"].(string); !ok || code != graphql.ErrJsonDecode {
			t.Errorf("expected error code %q, got %v", graphql.ErrJsonDecode, code)
		}
	})
}

// TestError_GetCode tests the GetCode helper method.
func TestError_GetCode(t *testing.T) {
	t.Run("returns code when present", func(t *testing.T) {
		err := graphql.Error{
			Message: "test error",
			Extensions: map[string]any{
				"code": graphql.ErrRequestError,
			},
		}

		got := err.GetCode()
		if got != graphql.ErrRequestError {
			t.Errorf("expected code %q, got %q", graphql.ErrRequestError, got)
		}
	})

	t.Run("returns empty string when extensions is nil", func(t *testing.T) {
		err := graphql.Error{
			Message: "test error",
		}

		got := err.GetCode()
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("returns empty string when code not present", func(t *testing.T) {
		err := graphql.Error{
			Message: "test error",
			Extensions: map[string]any{
				"other": "value",
			},
		}

		got := err.GetCode()
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("returns empty string when code is wrong type", func(t *testing.T) {
		err := graphql.Error{
			Message: "test error",
			Extensions: map[string]any{
				"code": 123,
			},
		}

		got := err.GetCode()
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})
}

// TestError_GetInternalExtensions tests the GetInternalExtensions
// helper method.
func TestError_GetInternalExtensions(t *testing.T) {
	t.Run("returns nil when extensions is nil", func(t *testing.T) {
		err := graphql.Error{
			Message: "test error",
		}

		got := err.GetInternalExtensions()
		if got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})

	t.Run("returns nil when internal not present", func(t *testing.T) {
		err := graphql.Error{
			Message: "test error",
			Extensions: map[string]any{
				"code": graphql.ErrRequestError,
			},
		}

		got := err.GetInternalExtensions()
		if got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})

	t.Run("returns typed request info", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": []string{"application/json"},
		}
		body := `{"query":"test"}`

		err := graphql.Error{
			Message: "test error",
			Extensions: map[string]any{
				"internal": map[string]any{
					"request": map[string]any{
						"headers": headers,
						"body":    body,
					},
				},
			},
		}

		got := err.GetInternalExtensions()
		if got == nil {
			t.Fatal("expected non-nil internal extensions")
		}

		if got.Request == nil {
			t.Fatal("expected non-nil request info")
		}

		if got.Request.Body != body {
			t.Errorf("expected body %q, got %q", body, got.Request.Body)
		}

		if len(got.Request.Headers) != len(headers) {
			t.Errorf(
				"expected %d headers, got %d",
				len(headers),
				len(got.Request.Headers),
			)
		}
	})

	t.Run("returns typed response info", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": []string{"application/json"},
		}
		body := `{"data":null}`

		err := graphql.Error{
			Message: "test error",
			Extensions: map[string]any{
				"internal": map[string]any{
					"response": map[string]any{
						"headers": headers,
						"body":    body,
					},
				},
			},
		}

		got := err.GetInternalExtensions()
		if got == nil {
			t.Fatal("expected non-nil internal extensions")
		}

		if got.Response == nil {
			t.Fatal("expected non-nil response info")
		}

		if got.Response.Body != body {
			t.Errorf("expected body %q, got %q", body, got.Response.Body)
		}

		if len(got.Response.Headers) != len(headers) {
			t.Errorf(
				"expected %d headers, got %d",
				len(headers),
				len(got.Response.Headers),
			)
		}
	})

	t.Run("returns typed error info", func(t *testing.T) {
		testErr := fmt.Errorf("test error detail")

		err := graphql.Error{
			Message: "test error",
			Extensions: map[string]any{
				"internal": map[string]any{
					"error": testErr,
				},
			},
		}

		got := err.GetInternalExtensions()
		if got == nil {
			t.Fatal("expected non-nil internal extensions")
		}

		if got.Error == nil {
			t.Fatal("expected non-nil error")
		}

		if got.Error.Error() != testErr.Error() {
			t.Errorf(
				"expected error %q, got %q",
				testErr.Error(),
				got.Error.Error(),
			)
		}
	})

	t.Run("returns all info when present", func(t *testing.T) {
		reqHeaders := http.Header{
			"Content-Type": []string{"application/json"},
		}
		reqBody := `{"query":"test"}`
		respHeaders := http.Header{
			"Content-Type": []string{"application/json"},
		}
		respBody := `{"data":null}`
		testErr := fmt.Errorf("io error")

		err := graphql.Error{
			Message: "test error",
			Extensions: map[string]any{
				"code": graphql.ErrRequestError,
				"internal": map[string]any{
					"request": map[string]any{
						"headers": reqHeaders,
						"body":    reqBody,
					},
					"response": map[string]any{
						"headers": respHeaders,
						"body":    respBody,
					},
					"error": testErr,
				},
			},
		}

		got := err.GetInternalExtensions()
		if got == nil {
			t.Fatal("expected non-nil internal extensions")
		}

		if got.Request == nil {
			t.Fatal("expected non-nil request info")
		}
		if got.Request.Body != reqBody {
			t.Errorf("expected request body %q, got %q", reqBody, got.Request.Body)
		}

		if got.Response == nil {
			t.Fatal("expected non-nil response info")
		}
		if got.Response.Body != respBody {
			t.Errorf(
				"expected response body %q, got %q",
				respBody,
				got.Response.Body,
			)
		}

		if got.Error == nil {
			t.Fatal("expected non-nil error")
		}
		if got.Error.Error() != testErr.Error() {
			t.Errorf(
				"expected error %q, got %q",
				testErr.Error(),
				got.Error.Error(),
			)
		}
	})

	t.Run("handles missing fields gracefully", func(t *testing.T) {
		err := graphql.Error{
			Message: "test error",
			Extensions: map[string]any{
				"internal": map[string]any{
					"request": map[string]any{
						// missing headers and body
					},
				},
			},
		}

		got := err.GetInternalExtensions()
		if got == nil {
			t.Fatal("expected non-nil internal extensions")
		}

		if got.Request == nil {
			t.Fatal("expected non-nil request info")
		}

		if got.Request.Body != "" {
			t.Errorf("expected empty body, got %q", got.Request.Body)
		}

		if got.Request.Headers != nil {
			t.Errorf("expected nil headers, got %+v", got.Request.Headers)
		}
	})

	t.Run("integration with DecorateError", func(t *testing.T) {
		// Test that errors decorated via DecorateError can be retrieved
		// via GetInternalExtensions
		client := graphql.NewClient("http://example.com", nil).WithDebug(true)

		reqBody := `{"query":"{test}"}`
		respBody := `{"data":null,"errors":[{"message":"error"}]}`
		req, err := http.NewRequest(http.MethodPost, "http://example.com", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp := &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}

		baseErr := graphql.Error{
			Message: "test error",
			Extensions: map[string]any{
				"code": graphql.ErrRequestError,
			},
		}

		decorated := client.DecorateError(
			baseErr,
			req,
			resp,
			strings.NewReader(reqBody),
			strings.NewReader(respBody),
		)

		// Test GetCode()
		code := decorated.GetCode()
		if code != graphql.ErrRequestError {
			t.Errorf("expected code %q, got %q", graphql.ErrRequestError, code)
		}

		// Test GetInternalExtensions()
		internal := decorated.GetInternalExtensions()
		if internal == nil {
			t.Fatal("expected non-nil internal extensions")
		}

		// Verify request info
		if internal.Request == nil {
			t.Fatal("expected non-nil request info")
		}
		if internal.Request.Body != reqBody {
			t.Errorf("expected request body %q, got %q", reqBody, internal.Request.Body)
		}
		if len(internal.Request.Headers) == 0 {
			t.Error("expected non-empty request headers")
		}

		// Verify response info
		if internal.Response == nil {
			t.Fatal("expected non-nil response info")
		}
		if internal.Response.Body != respBody {
			t.Errorf(
				"expected response body %q, got %q",
				respBody,
				internal.Response.Body,
			)
		}
		if len(internal.Response.Headers) == 0 {
			t.Error("expected non-empty response headers")
		}
	})
}

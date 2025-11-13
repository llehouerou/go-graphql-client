package graphql_test

import (
	"context"
	"encoding/json"
	"errors"
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

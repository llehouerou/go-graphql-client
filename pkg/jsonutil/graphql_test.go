package jsonutil_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/llehouerou/go-graphql-client/pkg/jsonutil"
)

func TestUnmarshalGraphQL(t *testing.T) {
	/*
		query {
			me {
				name
				height
			}
		}
	*/
	type query struct {
		Me struct {
			Name   string
			Height float64
		}
	}
	var got query
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"me": {
			"name": "Luke Skywalker",
			"height": 1.72
		}
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	var want query
	want.Me.Name = "Luke Skywalker"
	want.Me.Height = 1.72
	if !reflect.DeepEqual(got, want) {
		t.Error("not equal")
	}
}

func TestUnmarshalGraphQL_graphqlTag(t *testing.T) {
	type query struct {
		Foo string `graphql:"baz"`
	}
	var got query
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"baz": "bar"
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	want := query{
		Foo: "bar",
	}
	if !reflect.DeepEqual(got, want) {
		t.Error("not equal")
	}
}

func TestUnmarshalGraphQL_jsonTag(t *testing.T) {
	type query struct {
		Foo string `json:"baz"`
	}
	var got query
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"foo": "bar"
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	want := query{
		Foo: "bar",
	}
	if !reflect.DeepEqual(got, want) {
		t.Error("not equal")
	}
}

func TestUnmarshalGraphQL_jsonRawTag(t *testing.T) {
	type query struct {
		Data    json.RawMessage
		Another string
	}
	var got query
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"Data": { "foo":"bar" },
		"Another" : "stuff"
        }`), &got)

	if err != nil {
		t.Fatal(err)
	}
	want := query{
		Another: "stuff",
		Data:    []byte(`{"foo":"bar"}`),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("not equal: %v %v", want, got)
	}
}

func TestUnmarshalGraphQL_fieldAsScalar(t *testing.T) {
	type query struct {
		Data    json.RawMessage  `scalar:"true"`
		DataPtr *json.RawMessage `scalar:"true"`
		Another string
		Tags    map[string]int `scalar:"true"`
	}
	var got query
	err := jsonutil.UnmarshalGraphQL([]byte(`{
                "Data" : {"ValA":1,"ValB":"foo"},
                "DataPtr" : {"ValC":3,"ValD":false},
		"Another" : "stuff",
                "Tags": {
                    "keyA": 2,
                    "keyB": 3
                }
        }`), &got)

	if err != nil {
		t.Fatal(err)
	}
	dataPtr := json.RawMessage(`{"ValC":3,"ValD":false}`)
	want := query{
		Data:    json.RawMessage(`{"ValA":1,"ValB":"foo"}`),
		DataPtr: &dataPtr,
		Another: "stuff",
		Tags: map[string]int{
			"keyA": 2,
			"keyB": 3,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("not equal: %v %v", want, got)
	}
}

func TestUnmarshalGraphQL_orderedMap(t *testing.T) {
	type query [][2]any
	got := query{
		{"foo", ""},
	}
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"foo": "bar"
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	want := query{
		{"foo", "bar"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("not equal: %v != %v", got, want)
	}
}

func TestUnmarshalGraphQL_orderedMapWithPointers(t *testing.T) {
	// Test case similar to sorarezone usage - pointers in ordered map
	type GameFormation struct {
		Name string `graphql:"name"`
		ID   string `graphql:"id"`
	}

	game1 := &GameFormation{}
	game2 := &GameFormation{}

	got := [][2]any{
		{"game0:game(id:\"1\")", game1},
		{"game1:game(id:\"2\")", game2},
	}

	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"game0": {
			"name": "Game One",
			"id": "1"
		},
		"game1": {
			"name": "Game Two",
			"id": "2"
		}
	}`), &got)

	if err != nil {
		t.Fatal(err)
	}

	if game1.Name != "Game One" {
		t.Errorf("game1.Name = %q, want %q", game1.Name, "Game One")
	}
	if game1.ID != "1" {
		t.Errorf("game1.ID = %q, want %q", game1.ID, "1")
	}
	if game2.Name != "Game Two" {
		t.Errorf("game2.Name = %q, want %q", game2.Name, "Game Two")
	}
	if game2.ID != "2" {
		t.Errorf("game2.ID = %q, want %q", game2.ID, "2")
	}
}

func TestUnmarshalGraphQL_orderedMapAlias(t *testing.T) {
	type Update struct {
		Name string `graphql:"name"`
	}
	got := [][2]any{
		{"update0:update(name:$name0)", &Update{}},
		{"update1:update(name:$name1)", &Update{}},
	}
	err := jsonutil.UnmarshalGraphQL([]byte(`{
      "update0": {
        "name": "grihabor"
      },
      "update1": {
        "name": "diman"
      }
}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]any{
		{"update0:update(name:$name0)", &Update{Name: "grihabor"}},
		{"update1:update(name:$name1)", &Update{Name: "diman"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("not equal: %v != %v", got, want)
	}
}

func TestUnmarshalGraphQL_array(t *testing.T) {
	type query struct {
		Foo []string
		Bar []string
		Baz []string
	}
	var got query
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"foo": [
			"bar",
			"baz"
		],
		"bar": [],
		"baz": null
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	want := query{
		Foo: []string{"bar", "baz"},
		Bar: []string{},
		Baz: []string(nil),
	}
	if !reflect.DeepEqual(got, want) {
		t.Error("not equal")
	}
}

// When unmarshaling into an array, its initial value should be overwritten
// (rather than appended to).
func TestUnmarshalGraphQL_arrayReset(t *testing.T) {
	var got = []string{"initial"}
	err := jsonutil.UnmarshalGraphQL([]byte(`["bar", "baz"]`), &got)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"bar", "baz"}
	if !reflect.DeepEqual(got, want) {
		t.Error("not equal")
	}
}

func TestUnmarshalGraphQL_objectArray(t *testing.T) {
	type query struct {
		Foo []struct {
			Name string
		}
	}
	var got query
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"foo": [
			{"name": "bar"},
			{"name": "baz"}
		]
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	want := query{
		Foo: []struct{ Name string }{
			{"bar"},
			{"baz"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Error("not equal")
	}
}

func TestUnmarshalGraphQL_orderedMapArray(t *testing.T) {
	type query struct {
		Foo [][][2]any
	}
	got := query{
		Foo: [][][2]any{
			{{"name", ""}},
		},
	}
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"foo": [
			{"name": "bar"},
			{"name": "baz"}
		]
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	want := query{
		Foo: [][][2]any{
			{{"name", "bar"}},
			{{"name", "baz"}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Error("not equal")
	}
}

func TestUnmarshalGraphQL_pointer(t *testing.T) {
	s := "will be overwritten"
	foo := "foo"
	type query struct {
		Foo *string
		Bar *string
	}
	var got query
	got.Bar = &s // Test that got.Bar gets set to nil.
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"foo": "foo",
		"bar": null
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	want := query{
		Foo: &foo,
		Bar: nil,
	}
	if !reflect.DeepEqual(got, want) {
		t.Error("not equal")
	}
}

func TestUnmarshalGraphQL_objectPointerArray(t *testing.T) {
	bar := "bar"
	baz := "baz"
	type query struct {
		Foo []*struct {
			Name *string
		}
	}
	var got query
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"foo": [
			{"name": "bar"},
			null,
			{"name": "baz"}
		]
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	want := query{
		Foo: []*struct{ Name *string }{
			{&bar},
			nil,
			{&baz},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Error("not equal")
	}
}

func TestUnmarshalGraphQL_orderedMapNullInArray(t *testing.T) {
	type query struct {
		Foo [][][2]any
	}
	got := query{
		Foo: [][][2]any{
			{{"name", ""}},
		},
	}
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"foo": [
			{"name": "bar"},
			null,
			{"name": "baz"}
		]
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	want := query{
		Foo: [][][2]any{
			{{"name", "bar"}},
			nil,
			{{"name", "baz"}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Error("not equal")
	}
}

func TestUnmarshalGraphQL_pointerWithInlineFragment(t *testing.T) {
	type actor struct {
		User struct {
			DatabaseID uint64
		} `graphql:"... on User"`
		Login string
	}
	type query struct {
		Author actor
		Editor *actor
	}
	var got query
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"author": {
			"databaseId": 1,
			"login": "test1"
		},
		"editor": {
			"databaseId": 2,
			"login": "test2"
		}
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	var want query
	want.Author = actor{
		User:  struct{ DatabaseID uint64 }{1},
		Login: "test1",
	}
	want.Editor = &actor{
		User:  struct{ DatabaseID uint64 }{2},
		Login: "test2",
	}

	if !reflect.DeepEqual(got, want) {
		t.Error("not equal")
	}
}

func TestUnmarshalGraphQL_unexportedField(t *testing.T) {
	type query struct {
		foo *string //nolint:unused // Testing unexported field handling
	}
	err := jsonutil.UnmarshalGraphQL([]byte(`{"foo": "bar"}`), new(query))
	if err == nil {
		t.Fatal("got error: nil, want: non-nil")
	}
	if got, want := err.Error(), "struct field for \"foo\" doesn't exist in any of 1 places to unmarshal"; got != want {
		t.Errorf("got error: %v, want: %v", got, want)
	}
}

func TestUnmarshalGraphQL_multipleValues(t *testing.T) {
	type query struct {
		Foo *string
	}
	err := jsonutil.UnmarshalGraphQL([]byte(`{"foo": "bar"}{"foo": "baz"}`), new(query))
	if err == nil {
		t.Fatal("got error: nil, want: non-nil")
	}
	if got, want := err.Error(), "invalid token '{' after top-level value"; got != want {
		t.Errorf("got error: %v, want: %v", got, want)
	}
}

func TestUnmarshalGraphQL_multipleValuesInOrderedMap(t *testing.T) {
	type query [][2]any
	q := query{{"foo", ""}}
	err := jsonutil.UnmarshalGraphQL([]byte(`{"foo": "bar"}{"foo": "baz"}`), &q)
	if err == nil {
		t.Fatal("got error: nil, want: non-nil")
	}
	if got, want := err.Error(), "invalid token '{' after top-level value"; got != want {
		t.Errorf("got error: %v, want: %v", got, want)
	}
}

func TestUnmarshalGraphQL_union(t *testing.T) {
	/*
		{
			__typename
			... on ClosedEvent {
				createdAt
				actor {login}
			}
			... on ReopenedEvent {
				createdAt
				actor {login}
			}
		}
	*/
	type actor struct{ Login string }
	type closedEvent struct {
		Actor     actor
		CreatedAt time.Time
	}
	type reopenedEvent struct {
		Actor     actor
		CreatedAt time.Time
	}
	type issueTimelineItem struct {
		Typename      string        `graphql:"__typename"`
		ClosedEvent   closedEvent   `graphql:"... on ClosedEvent"`
		ReopenedEvent reopenedEvent `graphql:"... on ReopenedEvent"`
	}
	var got issueTimelineItem
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"__typename": "ClosedEvent",
		"createdAt": "2017-06-29T04:12:01Z",
		"actor": {
			"login": "shurcooL-test"
		}
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	want := issueTimelineItem{
		Typename: "ClosedEvent",
		ClosedEvent: closedEvent{
			Actor: actor{
				Login: "shurcooL-test",
			},
			CreatedAt: time.Unix(1498709521, 0).UTC(),
		},
		// ReopenedEvent should NOT be populated since __typename is "ClosedEvent"
		ReopenedEvent: reopenedEvent{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Error("not equal")
	}
}

func TestUnmarshalGraphQL_orderedMapUnion(t *testing.T) {
	/*
		{
			__typename
			... on ClosedEvent {
				createdAt
				actor {login}
			}
			... on ReopenedEvent {
				createdAt
				actor {login}
			}
		}
	*/
	closedEventActor := [][2]any{{"login", ""}}
	reopenedEventActor := [][2]any{{"login", ""}}
	closedEvent := [][2]any{{"actor", closedEventActor}, {"createdAt", time.Time{}}}
	reopenedEvent := [][2]any{{"actor", reopenedEventActor}, {"createdAt", time.Time{}}}
	got := [][2]any{
		{"__typename", ""},
		{"... on ClosedEvent", closedEvent},
		{"... on ReopenedEvent", reopenedEvent},
	}
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"__typename": "ClosedEvent",
		"createdAt": "2017-06-29T04:12:01Z",
		"actor": {
			"login": "shurcooL-test"
		}
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]any{
		{"__typename", "ClosedEvent"},
		{"... on ClosedEvent", [][2]any{
			{"actor", [][2]any{{"login", "shurcooL-test"}}},
			{"createdAt", time.Unix(1498709521, 0).UTC()},
		}},
		{"... on ReopenedEvent", [][2]any{
			{"actor", [][2]any{{"login", ""}}},
			{"createdAt", time.Time{}},
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("not equal:\ngot: %v\nwant: %v", got, want)
		createdAt := got[1][1].([][2]any)[1]
		t.Logf("key: %s, type: %v", createdAt[0], reflect.TypeOf(createdAt[1]))
	}
}

// Issue https://github.com/shurcooL/githubv4/issues/18.
func TestUnmarshalGraphQL_arrayInsideInlineFragment(t *testing.T) {
	/*
		query {
			search(type: ISSUE, first: 1, query: "type:pr repo:owner/name") {
				nodes {
					... on PullRequest {
						commits(last: 1) {
							nodes {
								url
							}
						}
					}
				}
			}
		}
	*/
	type query struct {
		Search struct {
			Nodes []struct {
				PullRequest struct {
					Commits struct {
						Nodes []struct {
							URL string `graphql:"url"`
						}
					} `graphql:"commits(last: 1)"`
				} `graphql:"... on PullRequest"`
			}
		} `graphql:"search(type: ISSUE, first: 1, query: \"type:pr repo:owner/name\")"`
	}
	var got query
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"search": {
			"nodes": [
				{
					"commits": {
						"nodes": [
							{
								"url": "https://example.org/commit/49e1"
							}
						]
					}
				}
			]
		}
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	var want query
	want.Search.Nodes = make([]struct {
		PullRequest struct {
			Commits struct {
				Nodes []struct {
					URL string `graphql:"url"`
				}
			} `graphql:"commits(last: 1)"`
		} `graphql:"... on PullRequest"`
	}, 1)
	want.Search.Nodes[0].PullRequest.Commits.Nodes = make([]struct {
		URL string `graphql:"url"`
	}, 1)
	want.Search.Nodes[0].PullRequest.Commits.Nodes[0].URL = "https://example.org/commit/49e1"
	if !reflect.DeepEqual(got, want) {
		t.Error("not equal")
	}
}

func TestUnmarshalGraphQL_unionWithConflictingFieldTypes(t *testing.T) {
	/*
		Issue: When a union type has inline fragments with fields of the same name
		but different types, unmarshaling fails with "cannot unmarshal string into
		Go value of type int" because the library tries to unmarshal all fields into
		ALL fragments instead of only the fragment matching __typename.

		GraphQL Query:
		{
			authorizations {
				__typename
				... on StarkexTransferAuthorizationRequest {
					nonce
					amount
				}
				... on SolanaTokenTransferAuthorizationRequest {
					nonce
					assetId
				}
				... on MangopayWalletTransferAuthorizationRequest {
					nonce
					amount
				}
			}
		}
	*/

	type starkexTransfer struct {
		Nonce  int    `graphql:"nonce"`  // int type
		Amount string `graphql:"amount"`
	}

	type solanaTokenTransfer struct {
		Nonce   string `graphql:"nonce"` // string type - CONFLICT!
		AssetId string `graphql:"assetId"`
	}

	type mangopayWalletTransfer struct {
		Nonce  int `graphql:"nonce"` // int type
		Amount int `graphql:"amount"`
	}

	type authorizationRequest struct {
		Typename               string                 `graphql:"__typename"`
		StarkexTransfer        starkexTransfer        `graphql:"... on StarkexTransferAuthorizationRequest"`
		SolanaTokenTransfer    solanaTokenTransfer    `graphql:"... on SolanaTokenTransferAuthorizationRequest"`
		MangopayWalletTransfer mangopayWalletTransfer `graphql:"... on MangopayWalletTransferAuthorizationRequest"`
	}

	var got authorizationRequest
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"__typename": "SolanaTokenTransferAuthorizationRequest",
		"nonce": "1234567890",
		"assetId": "0x123abc"
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	// Expected: Only the SolanaTokenTransfer fragment should be populated
	// since __typename matches "SolanaTokenTransferAuthorizationRequest"
	want := authorizationRequest{
		Typename: "SolanaTokenTransferAuthorizationRequest",
		SolanaTokenTransfer: solanaTokenTransfer{
			Nonce:   "1234567890",
			AssetId: "0x123abc",
		},
		// Other fragments should remain zero-valued
		StarkexTransfer:        starkexTransfer{},
		MangopayWalletTransfer: mangopayWalletTransfer{},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("not equal\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestUnmarshalGraphQL_unionWithoutTypename(t *testing.T) {
	/*
		Test backward compatibility: when there's no __typename field,
		all fragments should be populated (old behavior).
	*/

	type typeA struct {
		FieldA string `graphql:"fieldA"`
	}

	type typeB struct {
		FieldB int `graphql:"fieldB"`
	}

	type unionType struct {
		FragmentA typeA `graphql:"... on TypeA"`
		FragmentB typeB `graphql:"... on TypeB"`
	}

	var got unionType
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"fieldA": "value_a",
		"fieldB": 42
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	// Without __typename, BOTH fragments should be populated (backward compatibility)
	want := unionType{
		FragmentA: typeA{
			FieldA: "value_a",
		},
		FragmentB: typeB{
			FieldB: 42,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("not equal\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestUnmarshalGraphQL_interfaceFragment(t *testing.T) {
	/*
		Tests that interface fragments work correctly when __typename is a concrete
		type that implements the interface.

		GraphQL Query:
		{
			team {
				__typename
				... on TeamInterface {
					slug
				}
			}
		}

		When __typename is "Club" or "NationalTeam" (concrete types implementing
		TeamInterface), the slug field from the interface fragment should still
		be populated.
	*/

	type team struct {
		Typename string `graphql:"__typename"`
		Team     struct {
			Slug string `graphql:"slug"`
		} `graphql:"... on TeamInterface"`
	}

	// Test with Club type
	var gotClub team
	err := jsonutil.UnmarshalGraphQL([]byte(`{
		"__typename": "Club",
		"slug": "barcelona"
	}`), &gotClub)
	if err != nil {
		t.Fatal(err)
	}

	wantClub := team{
		Typename: "Club",
		Team: struct {
			Slug string `graphql:"slug"`
		}{
			Slug: "barcelona",
		},
	}

	if !reflect.DeepEqual(gotClub, wantClub) {
		t.Errorf("Club: not equal\ngot:  %+v\nwant: %+v", gotClub, wantClub)
	}

	// Test with NationalTeam type
	var gotNationalTeam team
	err = jsonutil.UnmarshalGraphQL([]byte(`{
		"__typename": "NationalTeam",
		"slug": "france"
	}`), &gotNationalTeam)
	if err != nil {
		t.Fatal(err)
	}

	wantNationalTeam := team{
		Typename: "NationalTeam",
		Team: struct {
			Slug string `graphql:"slug"`
		}{
			Slug: "france",
		},
	}

	if !reflect.DeepEqual(gotNationalTeam, wantNationalTeam) {
		t.Errorf("NationalTeam: not equal\ngot:  %+v\nwant: %+v", gotNationalTeam, wantNationalTeam)
	}
}

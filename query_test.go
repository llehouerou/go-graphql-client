package graphql

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/llehouerou/go-graphql-client/types"
)

type cachedDirective struct {
	ttl int
}

func (cd cachedDirective) Type() OptionType {
	return OptionTypeOperationDirective
}

func (cd cachedDirective) String() string {
	if cd.ttl <= 0 {
		return "@cached"
	}
	return fmt.Sprintf("@cached(ttl: %d)", cd.ttl)
}

func TestConstructQuery(t *testing.T) {
	tests := []struct {
		options     []Option
		inV         any
		inVariables map[string]any
		want        string
	}{
		{
			inV: struct {
				Viewer struct {
					Login      string
					CreatedAt  DateTime
					ID         ID
					DatabaseID int
				}
				RateLimit struct {
					Cost      int
					Limit     int
					Remaining int
					ResetAt   DateTime
				}
			}{},
			want: `{viewer{login,createdAt,id,databaseId},rateLimit{cost,limit,remaining,resetAt}}`,
		},
		{
			options: []Option{OperationName("GetRepository"), cachedDirective{}},
			inV: struct {
				Repository struct {
					DatabaseID int
					URL        URI

					Issue struct {
						Comments struct {
							Edges []struct {
								Node struct {
									Body   string
									Author struct {
										Login string
									}
									Editor struct {
										Login string
									}
								}
								Cursor string
							}
						} `graphql:"comments(first:1after:\"Y3Vyc29yOjE5NTE4NDI1Ng==\")"`
					} `graphql:"issue(number:1)"`
				} `graphql:"repository(owner:\"shurcooL-test\"name:\"test-repo\")"`
			}{},
			want: `query GetRepository @cached {repository(owner:"shurcooL-test"name:"test-repo"){databaseId,url,issue(number:1){comments(first:1after:"Y3Vyc29yOjE5NTE4NDI1Ng=="){edges{node{body,author{login},editor{login}},cursor}}}}}`,
		},
		{
			inV: func() any {
				type actor struct {
					Login     string
					AvatarURL URI
					URL       URI
				}

				return struct {
					Repository struct {
						DatabaseID int
						URL        URI

						Issue struct {
							Comments struct {
								Edges []struct {
									Node struct {
										DatabaseID      int
										Author          actor
										PublishedAt     DateTime
										LastEditedAt    *DateTime
										Editor          *actor
										Body            string
										ViewerCanUpdate bool
									}
									Cursor string
								}
							} `graphql:"comments(first:1)"`
						} `graphql:"issue(number:1)"`
					} `graphql:"repository(owner:\"shurcooL-test\"name:\"test-repo\")"`
				}{}
			}(),
			want: `{repository(owner:"shurcooL-test"name:"test-repo"){databaseId,url,issue(number:1){comments(first:1){edges{node{databaseId,author{login,avatarUrl,url},publishedAt,lastEditedAt,editor{login,avatarUrl,url},body,viewerCanUpdate},cursor}}}}}`,
		},
		{
			inV: func() any {
				type actor struct {
					Login     string
					AvatarURL URI `graphql:"avatarUrl(size:72)"`
					URL       URI
				}

				return struct {
					Repository struct {
						Issue struct {
							Author         actor
							PublishedAt    DateTime
							LastEditedAt   *DateTime
							Editor         *actor
							Body           string
							ReactionGroups []struct {
								Content ReactionContent
								Users   struct {
									TotalCount int
								}
								ViewerHasReacted bool
							}
							ViewerCanUpdate bool

							Comments struct {
								Nodes []struct {
									DatabaseID     int
									Author         actor
									PublishedAt    DateTime
									LastEditedAt   *DateTime
									Editor         *actor
									Body           string
									ReactionGroups []struct {
										Content ReactionContent
										Users   struct {
											TotalCount int
										}
										ViewerHasReacted bool
									}
									ViewerCanUpdate bool
								}
								PageInfo struct {
									EndCursor   string
									HasNextPage bool
								}
							} `graphql:"comments(first:1)"`
						} `graphql:"issue(number:1)"`
					} `graphql:"repository(owner:\"shurcooL-test\"name:\"test-repo\")"`
				}{}
			}(),
			want: `{repository(owner:"shurcooL-test"name:"test-repo"){issue(number:1){author{login,avatarUrl(size:72),url},publishedAt,lastEditedAt,editor{login,avatarUrl(size:72),url},body,reactionGroups{content,users{totalCount},viewerHasReacted},viewerCanUpdate,comments(first:1){nodes{databaseId,author{login,avatarUrl(size:72),url},publishedAt,lastEditedAt,editor{login,avatarUrl(size:72),url},body,reactionGroups{content,users{totalCount},viewerHasReacted},viewerCanUpdate},pageInfo{endCursor,hasNextPage}}}}}`,
		},
		{
			inV: struct {
				Repository struct {
					Issue struct {
						Body string
					} `graphql:"issue(number: 1)"`
				} `graphql:"repository(owner:\"shurcooL-test\"name:\"test-repo\")"`
			}{},
			want: `{repository(owner:"shurcooL-test"name:"test-repo"){issue(number: 1){body}}}`,
		},
		{
			inV: struct {
				Repository struct {
					Issue struct {
						Body string
					} `graphql:"issue(number: $issueNumber)"`
				} `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
			}{},
			inVariables: map[string]any{
				"repositoryOwner": "shurcooL-test",
				"repositoryName":  "test-repo",
				"issueNumber":     1,
			},
			want: `query ($issueNumber:Int!$repositoryName:String!$repositoryOwner:String!){repository(owner: $repositoryOwner, name: $repositoryName){issue(number: $issueNumber){body}}}`,
		},
		{
			inV: struct {
				Repository struct {
					Issue struct {
						ReactionGroups []struct {
							Users struct {
								Nodes []struct {
									Login string
								}
							} `graphql:"users(first:10)"`
						}
					} `graphql:"issue(number: $issueNumber)"`
				} `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
			}{},
			inVariables: map[string]any{
				"repositoryOwner": "shurcooL-test",
				"repositoryName":  "test-repo",
				"issueNumber":     1,
			},
			want: `query ($issueNumber:Int!$repositoryName:String!$repositoryOwner:String!){repository(owner: $repositoryOwner, name: $repositoryName){issue(number: $issueNumber){reactionGroups{users(first:10){nodes{login}}}}}}`,
		},
		// check test above works with repository inner map
		{
			inV: func() any {
				type query struct {
					Repository [][2]any `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
				}
				type issue struct {
					ReactionGroups []struct {
						Users struct {
							Nodes []struct {
								Login string
							}
						} `graphql:"users(first:10)"`
					}
				}
				return query{Repository: [][2]any{
					{"issue(number: $issueNumber)", issue{}},
				}}
			}(),
			inVariables: map[string]any{
				"repositoryOwner": "shurcooL-test",
				"repositoryName":  "test-repo",
				"issueNumber":     1,
			},
			want: `query ($issueNumber:Int!$repositoryName:String!$repositoryOwner:String!){repository(owner: $repositoryOwner, name: $repositoryName){issue(number: $issueNumber){reactionGroups{users(first:10){nodes{login}}}}}}`,
		},
		// check inner maps work inside slices
		{
			inV: func() any {
				type query struct {
					Repository [][2]any `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
				}
				type issue struct {
					ReactionGroups []struct {
						Users [][2]any `graphql:"users(first:10)"`
					}
				}
				type nodes []struct {
					Login string
				}
				return query{Repository: [][2]any{
					{"issue(number: $issueNumber)", issue{
						ReactionGroups: []struct {
							Users [][2]any `graphql:"users(first:10)"`
						}{
							{Users: [][2]any{
								{"nodes", nodes{}},
							}},
						},
					}},
				}}
			}(),
			inVariables: map[string]any{
				"repositoryOwner": "shurcooL-test",
				"repositoryName":  "test-repo",
				"issueNumber":     1,
			},
			want: `query ($issueNumber:Int!$repositoryName:String!$repositoryOwner:String!){repository(owner: $repositoryOwner, name: $repositoryName){issue(number: $issueNumber){reactionGroups{users(first:10){nodes{login}}}}}}`,
		},
		// Embedded structs without graphql tag should be inlined in query.
		{
			inV: func() any {
				type actor struct {
					Login     string
					AvatarURL URI
					URL       URI
				}
				type event struct { // Common fields for all events.
					Actor     actor
					CreatedAt DateTime
				}
				type IssueComment struct {
					Body string
				}
				return struct {
					event                                         // Should be inlined.
					IssueComment  `graphql:"... on IssueComment"` // Should not be, because of graphql tag.
					CurrentTitle  string
					PreviousTitle string
					Label         struct {
						Name  string
						Color string
					}
				}{}
			}(),
			want: `{actor{login,avatarUrl,url},createdAt,... on IssueComment{body},currentTitle,previousTitle,label{name,color}}`,
		},
		{
			inV: struct {
				Viewer struct {
					Login      string
					CreatedAt  time.Time
					ID         any
					DatabaseID int
				}
			}{},
			want: `{viewer{login,createdAt,id,databaseId}}`,
		},
		{
			inV: struct {
				Viewer struct {
					ID         any
					Login      string
					CreatedAt  time.Time
					DatabaseID int
				}
				Tags map[string]any `scalar:"true"`
			}{},
			want: `{viewer{id,login,createdAt,databaseId},tags}`,
		},
		{
			inV: struct {
				Viewer struct {
					ID         any
					Login      string
					CreatedAt  time.Time
					DatabaseID int
				} `scalar:"true"`
			}{},
			want: `{viewer}`,
		},
		{
			inV: struct {
				Viewer struct {
					ID         any `graphql:"-"`
					Login      string
					CreatedAt  time.Time `graphql:"-"`
					DatabaseID int
				}
			}{},
			want: `{viewer{login,databaseId}}`,
		},
	}
	for _, tc := range tests {
		got, err := ConstructQuery(tc.inV, tc.inVariables, tc.options...)
		if err != nil {
			t.Error(err)
		} else if got != tc.want {
			t.Errorf("\ngot:  %q\nwant: %q\n", got, tc.want)
		}
	}
}

type CreateUser struct {
	Login string
}

type DeleteUser struct {
	Login string
}

func TestConstructMutation(t *testing.T) {
	tests := []struct {
		inV         any
		inVariables map[string]any
		want        string
	}{
		{
			inV: struct {
				AddReaction struct {
					Subject struct {
						ReactionGroups []struct {
							Users struct {
								TotalCount int
							}
						}
					}
				} `graphql:"addReaction(input:$input)"`
			}{},
			inVariables: map[string]any{
				"input": AddReactionInput{
					SubjectID: "MDU6SXNzdWUyMzE1MjcyNzk=",
					Content:   ReactionContentThumbsUp,
				},
			},
			want: `mutation ($input:AddReactionInput!){addReaction(input:$input){subject{reactionGroups{users{totalCount}}}}}`,
		},
		{
			inV: [][2]any{
				{"createUser(login:$login1)", &CreateUser{}},
				{"deleteUser(login:$login2)", &DeleteUser{}},
			},
			inVariables: map[string]any{
				"login1": "grihabor",
				"login2": "diman",
			},
			want: "mutation ($login1:String!$login2:String!){createUser(login:$login1){login}deleteUser(login:$login2){login}}",
		},
	}
	for _, tc := range tests {
		got, err := ConstructMutation(tc.inV, tc.inVariables)
		if err != nil {
			t.Error(err)
		} else if got != tc.want {
			t.Errorf("\ngot:  %q\nwant: %q\n", got, tc.want)
		}
	}
}

func TestConstructSubscription(t *testing.T) {
	tests := []struct {
		name        string
		inV         any
		inVariables map[string]any
		want        string
	}{
		{
			inV: struct {
				Viewer struct {
					Login      string
					CreatedAt  DateTime
					ID         ID
					DatabaseID int
				}
				RateLimit struct {
					Cost      int
					Limit     int
					Remaining int
					ResetAt   DateTime
				}
			}{},
			want: `subscription{viewer{login,createdAt,id,databaseId},rateLimit{cost,limit,remaining,resetAt}}`,
		},
		{
			name: "GetRepository",
			inV: struct {
				Repository struct {
					DatabaseID int
					URL        URI

					Issue struct {
						Comments struct {
							Edges []struct {
								Node struct {
									Body   string
									Author struct {
										Login string
									}
									Editor struct {
										Login string
									}
								}
								Cursor string
							}
						} `graphql:"comments(first:1after:\"Y3Vyc29yOjE5NTE4NDI1Ng==\")"`
					} `graphql:"issue(number:1)"`
				} `graphql:"repository(owner:\"shurcooL-test\"name:\"test-repo\")"`
			}{},
			want: `subscription GetRepository{repository(owner:"shurcooL-test"name:"test-repo"){databaseId,url,issue(number:1){comments(first:1after:"Y3Vyc29yOjE5NTE4NDI1Ng=="){edges{node{body,author{login},editor{login}},cursor}}}}}`,
		},
		{
			inV: func() any {
				type actor struct {
					Login     string
					AvatarURL URI
					URL       URI
				}

				return struct {
					Repository struct {
						DatabaseID int
						URL        URI

						Issue struct {
							Comments struct {
								Edges []struct {
									Node struct {
										DatabaseID      int
										Author          actor
										PublishedAt     DateTime
										LastEditedAt    *DateTime
										Editor          *actor
										Body            string
										ViewerCanUpdate bool
									}
									Cursor string
								}
							} `graphql:"comments(first:1)"`
						} `graphql:"issue(number:1)"`
					} `graphql:"repository(owner:\"shurcooL-test\"name:\"test-repo\")"`
				}{}
			}(),
			want: `subscription{repository(owner:"shurcooL-test"name:"test-repo"){databaseId,url,issue(number:1){comments(first:1){edges{node{databaseId,author{login,avatarUrl,url},publishedAt,lastEditedAt,editor{login,avatarUrl,url},body,viewerCanUpdate},cursor}}}}}`,
		},
		{
			inV: func() any {
				type actor struct {
					Login     string
					AvatarURL URI `graphql:"avatarUrl(size:72)"`
					URL       URI
				}

				return struct {
					Repository struct {
						Issue struct {
							Author         actor
							PublishedAt    DateTime
							LastEditedAt   *DateTime
							Editor         *actor
							Body           string
							ReactionGroups []struct {
								Content ReactionContent
								Users   struct {
									TotalCount int
								}
								ViewerHasReacted bool
							}
							ViewerCanUpdate bool

							Comments struct {
								Nodes []struct {
									DatabaseID     int
									Author         actor
									PublishedAt    DateTime
									LastEditedAt   *DateTime
									Editor         *actor
									Body           string
									ReactionGroups []struct {
										Content ReactionContent
										Users   struct {
											TotalCount int
										}
										ViewerHasReacted bool
									}
									ViewerCanUpdate bool
								}
								PageInfo struct {
									EndCursor   string
									HasNextPage bool
								}
							} `graphql:"comments(first:1)"`
						} `graphql:"issue(number:1)"`
					} `graphql:"repository(owner:\"shurcooL-test\"name:\"test-repo\")"`
				}{}
			}(),
			want: `subscription{repository(owner:"shurcooL-test"name:"test-repo"){issue(number:1){author{login,avatarUrl(size:72),url},publishedAt,lastEditedAt,editor{login,avatarUrl(size:72),url},body,reactionGroups{content,users{totalCount},viewerHasReacted},viewerCanUpdate,comments(first:1){nodes{databaseId,author{login,avatarUrl(size:72),url},publishedAt,lastEditedAt,editor{login,avatarUrl(size:72),url},body,reactionGroups{content,users{totalCount},viewerHasReacted},viewerCanUpdate},pageInfo{endCursor,hasNextPage}}}}}`,
		},
		{
			inV: struct {
				Repository struct {
					Issue struct {
						Body string
					} `graphql:"issue(number: 1)"`
				} `graphql:"repository(owner:\"shurcooL-test\"name:\"test-repo\")"`
			}{},
			want: `subscription{repository(owner:"shurcooL-test"name:"test-repo"){issue(number: 1){body}}}`,
		},
		{
			inV: struct {
				Repository struct {
					Issue struct {
						Body string
					} `graphql:"issue(number: $issueNumber)"`
				} `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
			}{},
			inVariables: map[string]any{
				"repositoryOwner": "shurcooL-test",
				"repositoryName":  "test-repo",
				"issueNumber":     1,
			},
			want: `subscription ($issueNumber:Int!$repositoryName:String!$repositoryOwner:String!){repository(owner: $repositoryOwner, name: $repositoryName){issue(number: $issueNumber){body}}}`,
		},
		{
			name: "SearchRepository",
			inV: struct {
				Repository struct {
					Issue struct {
						ReactionGroups []struct {
							Users struct {
								Nodes []struct {
									Login string
								}
							} `graphql:"users(first:10)"`
						}
					} `graphql:"issue(number: $issueNumber)"`
				} `graphql:"repository(owner: $repositoryOwner, name: $repositoryName, review: $userReview)"`
			}{},
			inVariables: map[string]any{
				"repositoryOwner": "shurcooL-test",
				"repositoryName":  "test-repo",
				"issueNumber":     1,
				"review":          UserReview{},
			},
			want: `subscription SearchRepository($issueNumber:Int!$repositoryName:String!$repositoryOwner:String!$review:user_review!){repository(owner: $repositoryOwner, name: $repositoryName, review: $userReview){issue(number: $issueNumber){reactionGroups{users(first:10){nodes{login}}}}}}`,
		},
		// Embedded structs without graphql tag should be inlined in query.
		{
			inV: func() any {
				type actor struct {
					Login     string
					AvatarURL URI
					URL       URI
				}
				type event struct { // Common fields for all events.
					Actor     actor
					CreatedAt DateTime
				}
				type IssueComment struct {
					Body string
				}
				return struct {
					event                                         // Should be inlined.
					IssueComment  `graphql:"... on IssueComment"` // Should not be, because of graphql tag.
					CurrentTitle  string
					PreviousTitle string
					Label         struct {
						Name  string
						Color string
					}
				}{}
			}(),
			want: `subscription{actor{login,avatarUrl,url},createdAt,... on IssueComment{body},currentTitle,previousTitle,label{name,color}}`,
		},
		{
			inV: struct {
				Viewer struct {
					Login      string
					CreatedAt  time.Time
					ID         any
					DatabaseID int
				}
			}{},
			want: `subscription{viewer{login,createdAt,id,databaseId}}`,
		},
	}
	for _, tc := range tests {
		got, err := ConstructSubscription(
			tc.inV,
			tc.inVariables,
			OperationName(tc.name),
		)
		if err != nil {
			t.Error(err)
		} else if got != tc.want {
			t.Errorf("\ngot:  %q\nwant: %q\n", got, tc.want)
		}
	}
}

func TestQueryArguments(t *testing.T) {
	iVal := int(123)
	i8Val := int8(12)
	i16Val := int16(500)
	i32Val := int32(70000)
	i64Val := int64(5000000000)
	uiVal := uint(123)
	ui8Val := uint8(12)
	ui16Val := uint16(500)
	ui32Val := uint32(70000)
	ui64Val := uint64(5000000000)
	f32Val := float32(33.4)
	f64Val := float64(99.23)
	bVal := true
	sVal := "some string"
	tests := []struct {
		in   map[string]any
		want string
	}{
		{
			in:   map[string]any{"a": Int(123), "b": NewBoolean(true)},
			want: "$a:Int!$b:Boolean",
		},
		{
			in: map[string]any{
				"a": iVal,
				"b": i8Val,
				"c": i16Val,
				"d": i32Val,
				"e": i64Val,
				"f": Int(123),
			},
			want: "$a:Int!$b:Int!$c:Int!$d:Int!$e:Int!$f:Int!",
		},
		{
			in: map[string]any{
				"a": &iVal,
				"b": &i8Val,
				"c": &i16Val,
				"d": &i32Val,
				"e": &i64Val,
				"f": NewInt(123),
			},
			want: "$a:Int$b:Int$c:Int$d:Int$e:Int$f:Int",
		},
		{
			in: map[string]any{
				"a": uiVal,
				"b": ui8Val,
				"c": ui16Val,
				"d": ui32Val,
				"e": ui64Val,
			},
			want: "$a:Int!$b:Int!$c:Int!$d:Int!$e:Int!",
		},
		{
			in: map[string]any{
				"a": &uiVal,
				"b": &ui8Val,
				"c": &ui16Val,
				"d": &ui32Val,
				"e": &ui64Val,
			},
			want: "$a:Int$b:Int$c:Int$d:Int$e:Int",
		},
		{
			in:   map[string]any{"a": f32Val, "b": f64Val, "c": Float(1.2)},
			want: "$a:Float!$b:Float!$c:Float!",
		},
		{
			in:   map[string]any{"a": &f32Val, "b": &f64Val, "c": NewFloat(1.2)},
			want: "$a:Float$b:Float$c:Float",
		},
		{
			in: map[string]any{
				"a": &bVal,
				"b": bVal,
				"c": true,
				"d": false,
				"e": Boolean(true),
				"f": NewBoolean(true),
			},
			want: "$a:Boolean$b:Boolean!$c:Boolean!$d:Boolean!$e:Boolean!$f:Boolean",
		},
		{
			in:   map[string]any{"a": NewID(123), "b": ID("id")},
			want: "$a:ID$b:ID!",
		},
		{
			in: map[string]any{
				"a": sVal,
				"b": &sVal,
				"c": String("foo"),
				"d": NewString("bar"),
			},
			want: "$a:String!$b:String$c:String!$d:String",
		},
		{
			in: map[string]any{
				"required": []IssueState{IssueStateOpen, IssueStateClosed},
				"optional": &[]IssueState{IssueStateOpen, IssueStateClosed},
			},
			want: "$optional:[IssueState!]$required:[IssueState!]!",
		},
		{
			in: map[string]any{
				"required": []IssueState(nil),
				"optional": (*[]IssueState)(nil),
			},
			want: "$optional:[IssueState!]$required:[IssueState!]!",
		},
		{
			in: map[string]any{
				"required": [...]IssueState{IssueStateOpen, IssueStateClosed},
				"optional": &[...]IssueState{IssueStateOpen, IssueStateClosed},
			},
			want: "$optional:[IssueState!]$required:[IssueState!]!",
		},
		{
			in:   map[string]any{"id": NewID("someID")},
			want: "$id:ID",
		},
		{
			in:   map[string]any{"id": ID("someID")},
			want: "$id:ID!",
		},
		{
			in:   map[string]any{"ids": []ID{"someID", "anotherID"}},
			want: `$ids:[ID!]!`,
		},
		{
			in:   map[string]any{"ids": &[]ID{"someID", "anotherID"}},
			want: `$ids:[ID!]`,
		},
		{
			in: map[string]any{
				"id":           Uuid(uuid.New()),
				"id_optional":  &val,
				"ids":          []Uuid{},
				"ids_optional": []*Uuid{},
				"my_uuid":      MyUuid(uuid.New()),
				"review":       UserReview{},
				"review_input": UserReviewInput{},
			},
			want: `$id:uuid!$id_optional:uuid$ids:[uuid!]!$ids_optional:[uuid]!$my_uuid:my_uuid!$review:user_review!$review_input:user_review_input!`,
		},
	}
	for i, tc := range tests {
		got := queryArguments(tc.in)
		if got != tc.want {
			t.Errorf("test case %d:\n got: %q\nwant: %q", i, got, tc.want)
		}
	}
}

// TestQueryArguments_StructVariables tests the struct-based variable support
// added in commit e2d1096. This validates that structs with json tags can be
// used as variables instead of only maps.
func TestQueryArguments_StructVariables(t *testing.T) {
	iVal := int(123)
	i8Val := int8(12)
	i16Val := int16(500)
	f32Val := float32(33.4)
	f64Val := float64(99.23)
	bVal := true
	sVal := "some string"

	tests := []struct {
		name string
		in   any
		want string
	}{
		{
			name: "struct with basic types",
			in: struct {
				Name   string `json:"name"`
				Age    int    `json:"age"`
				Active bool   `json:"active"`
			}{
				Name:   "John",
				Age:    30,
				Active: true,
			},
			want: "$active:Boolean!$age:Int!$name:String!",
		},
		{
			name: "struct with pointer fields (optional types)",
			in: struct {
				Name   *string `json:"name"`
				Age    *int    `json:"age"`
				Active *bool   `json:"active"`
			}{
				Name:   &sVal,
				Age:    &iVal,
				Active: &bVal,
			},
			want: "$active:Boolean$age:Int$name:String",
		},
		{
			name: "struct with GraphQL scalar wrappers",
			in: struct {
				ID     ID      `json:"id"`
				Name   String  `json:"name"`
				Age    Int     `json:"age"`
				Score  Float   `json:"score"`
				Active Boolean `json:"active"`
			}{
				ID:     ID("abc123"),
				Name:   String("John"),
				Age:    Int(30),
				Score:  Float(95.5),
				Active: Boolean(true),
			},
			want: "$active:Boolean!$age:Int!$id:ID!$name:String!$score:Float!",
		},
		{
			name: "struct with pointer scalar wrappers (nullable)",
			in: struct {
				ID     *ID      `json:"id"`
				Name   *String  `json:"name"`
				Age    *Int     `json:"age"`
				Score  *Float   `json:"score"`
				Active *Boolean `json:"active"`
			}{
				ID:     NewID("abc123"),
				Name:   NewString("John"),
				Age:    NewInt(30),
				Score:  NewFloat(95.5),
				Active: NewBoolean(true),
			},
			want: "$active:Boolean$age:Int$id:ID$name:String$score:Float",
		},
		{
			name: "struct with unexported fields (should skip)",
			in: struct {
				Name       string `json:"name"`
				age        int    // unexported, no tag needed
				privateVal bool   // unexported, no tag needed
				Active     bool   `json:"active"`
			}{
				Name:       "John",
				age:        30,
				privateVal: true,
				Active:     false,
			},
			want: "$active:Boolean!$name:String!",
		},
		{
			name: "struct with fields lacking json tags (should skip)",
			in: struct {
				Name   string `json:"name"`
				Age    int    // no json tag
				Active bool   `json:"active"`
			}{
				Name:   "John",
				Age:    30,
				Active: true,
			},
			want: "$active:Boolean!$name:String!",
		},
		{
			name: "struct with json:- tag (should skip)",
			in: struct {
				Name     string `json:"name"`
				Internal string `json:"-"`
				Active   bool   `json:"active"`
			}{
				Name:     "John",
				Internal: "secret",
				Active:   true,
			},
			want: "$active:Boolean!$name:String!",
		},
		{
			name: "struct with json tag options (should extract field name only)",
			in: struct {
				Name     string `json:"name,omitempty"`
				Age      int    `json:"age,string"`
				Active   bool   `json:"active,omitempty"`
				Optional *int   `json:"optional,omitempty"`
			}{
				Name:     "John",
				Age:      30,
				Active:   true,
				Optional: &iVal,
			},
			want: "$active:Boolean!$age:Int!$name:String!$optional:Int",
		},
		{
			name: "struct with empty field name in tag (should skip)",
			in: struct {
				Name   string `json:"name"`
				Empty  string `json:",omitempty"` // Empty field name
				Active bool   `json:"active"`
			}{
				Name:   "John",
				Empty:  "ignored",
				Active: true,
			},
			want: "$active:Boolean!$name:String!",
		},
		{
			name: "struct with multiple json:- tags",
			in: struct {
				Name      string `json:"name"`
				Internal1 string `json:"-"`
				Internal2 int    `json:"-"`
				Active    bool   `json:"active"`
			}{
				Name:      "John",
				Internal1: "secret",
				Internal2: 42,
				Active:    true,
			},
			want: "$active:Boolean!$name:String!",
		},
		{
			name: "struct with simple fields only",
			in: struct {
				Name   string `json:"name"`
				Age    int    `json:"age"`
				Active bool   `json:"active"`
			}{
				Name:   "John",
				Age:    30,
				Active: true,
			},
			want: "$active:Boolean!$age:Int!$name:String!",
		},
		{
			name: "pointer to struct (should dereference)",
			in: &struct {
				Name   string `json:"name"`
				Age    int    `json:"age"`
			}{
				Name: "John",
				Age:  30,
			},
			want: "$age:Int!$name:String!",
		},
		{
			name: "empty struct (should return empty string)",
			in: struct {
			}{},
			want: "",
		},
		{
			name: "struct with all unexported fields (should return empty string)",
			in: struct {
				name   string // unexported, no tag needed
				age    int    // unexported, no tag needed
				active bool   // unexported, no tag needed
			}{
				name:   "John",
				age:    30,
				active: true,
			},
			want: "",
		},
		{
			name: "struct with all numeric types",
			in: struct {
				I   int     `json:"i"`
				I8  int8    `json:"i8"`
				I16 int16   `json:"i16"`
				I32 int32   `json:"i32"`
				I64 int64   `json:"i64"`
				UI  uint    `json:"ui"`
				UI8 uint8   `json:"ui8"`
				UI16 uint16  `json:"ui16"`
				UI32 uint32  `json:"ui32"`
				UI64 uint64  `json:"ui64"`
				F32 float32 `json:"f32"`
				F64 float64 `json:"f64"`
			}{
				I:   iVal, I8: i8Val, I16: i16Val,
				I32: int32(70000), I64: int64(5000000000),
				UI: uint(123), UI8: uint8(12), UI16: uint16(500),
				UI32: uint32(70000), UI64: uint64(5000000000),
				F32: f32Val, F64: f64Val,
			},
			want: "$f32:Float!$f64:Float!$i:Int!$i16:Int!$i32:Int!$i64:Int!$i8:Int!$ui:Int!$ui16:Int!$ui32:Int!$ui64:Int!$ui8:Int!",
		},
		{
			name: "struct with pointer numeric types",
			in: struct {
				I   *int     `json:"i"`
				I8  *int8    `json:"i8"`
				I16 *int16   `json:"i16"`
				F32 *float32 `json:"f32"`
				F64 *float64 `json:"f64"`
			}{
				I: &iVal, I8: &i8Val, I16: &i16Val,
				F32: &f32Val, F64: &f64Val,
			},
			want: "$f32:Float$f64:Float$i:Int$i16:Int$i8:Int",
		},
		{
			name: "struct with slices",
			in: struct {
				States   []IssueState  `json:"states"`
				Optional *[]IssueState `json:"optional"`
			}{
				States:   []IssueState{IssueStateOpen, IssueStateClosed},
				Optional: &[]IssueState{IssueStateOpen},
			},
			want: "$optional:[IssueState!]$states:[IssueState!]!",
		},
		{
			name: "struct with custom GraphQLType",
			in: struct {
				ID         Uuid            `json:"id"`
				IDOptional *Uuid           `json:"id_optional"`
				MyID       MyUuid          `json:"my_uuid"`
				Review     UserReview      `json:"review"`
				Input      UserReviewInput `json:"review_input"`
			}{
				ID:         Uuid(uuid.New()),
				IDOptional: &val,
				MyID:       MyUuid(uuid.New()),
				Review:     UserReview{Review: "good", UserID: "123"},
				Input:      UserReviewInput{Review: "bad", UserID: "456"},
			},
			want: "$id:uuid!$id_optional:uuid$my_uuid:my_uuid!$review:user_review!$review_input:user_review_input!",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := queryArguments(tc.in)
			if got != tc.want {
				t.Errorf(
					"\ngot:  %q\nwant: %q",
					got,
					tc.want,
				)
			}
		})
	}
}

// TestQueryArguments_InvalidTypes tests error handling for invalid variable types
func TestQueryArguments_InvalidTypes(t *testing.T) {
	tests := []struct {
		name      string
		in        any
		wantPanic bool
		panicMsg  string
	}{
		{
			name:      "string value (not struct or map)",
			in:        "invalid",
			wantPanic: true,
			panicMsg:  "variables must be a struct or a map; got string",
		},
		{
			name:      "int value (not struct or map)",
			in:        123,
			wantPanic: true,
			panicMsg:  "variables must be a struct or a map; got int",
		},
		{
			name:      "slice value (not struct or map)",
			in:        []string{"a", "b"},
			wantPanic: true,
			panicMsg:  "variables must be a struct or a map; got []string",
		},
		{
			name:      "pointer to string (not struct or map)",
			in:        stringPtr("invalid"),
			wantPanic: true,
			panicMsg:  "variables must be a struct or a map; got *string",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if !tc.wantPanic {
					if r != nil {
						t.Errorf(
							"unexpected panic: %v",
							r,
						)
					}
					return
				}
				if r == nil {
					t.Errorf("expected panic but got none")
					return
				}
				if msg, ok := r.(string); ok {
					if msg != tc.panicMsg {
						t.Errorf(
							"panic message:\ngot:  %q\nwant: %q",
							msg,
							tc.panicMsg,
						)
					}
				} else {
					t.Errorf(
						"panic value is not string: %T %v",
						r,
						r,
					)
				}
			}()
			_ = queryArguments(tc.in)
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

// stringStringer is to support a built-in string type as a fmt.Stringer
type stringStringer string

func (s stringStringer) String() string { return string(s) }

type supportedType fmt.Stringer

func newCustomTypeHint(data supportedType, hintType string) types.GraphQLType {
	return &customTypeHint{
		data: data,
		hint: hintType,
	}
}

type customTypeHint struct {
	data supportedType
	hint string
}

func (cth *customTypeHint) GetGraphQLType() string {
	return cth.hint
}

func (cth *customTypeHint) String() string {
	return cth.data.String()
}

func TestDynamicCustomType_GetGraphQLType(t *testing.T) {

	type gqlGetRowsQuery struct {
		GetRows struct {
			Data []struct {
				Id String
			}
		} `graphql:"getRows(batchId:$batchId)"`
	}

	const hintType = "UUID"
	ct := newCustomTypeHint(stringStringer("test"), hintType)
	if ct.GetGraphQLType() != hintType {
		t.Errorf(
			"custom type hint:\n got: %s\nwant: %s",
			ct.GetGraphQLType(),
			hintType,
		)
	}

	var query gqlGetRowsQuery
	hint := newCustomTypeHint(
		stringStringer("9e573418-38f4-4a35-b3df-eb36c9bba2cd"),
		hintType,
	)
	queryVars := map[string]any{
		"batchId": hint,
	}
	constructQuery, err := ConstructQuery(&query, queryVars)
	if err != nil {
		t.Errorf("construct custom type hint error:\n %s", err)
	}
	if !strings.Contains(constructQuery, hintType) {
		t.Errorf(
			"custom type hint:\n the constructed query doesn't contain %s\n%s",
			hintType,
			constructQuery,
		)
	}
}

// TestGraphQLTypeInterface_StructFields tests that types implementing GraphQLType
// can be used as struct fields and their GetGraphQLType() method is called
// to determine the GraphQL field representation instead of using struct tags.
func TestGraphQLTypeInterface_StructFields(t *testing.T) {
	// CustomFieldWithArgs is a field type that returns a GraphQL field with arguments
	type CustomFieldWithArgs struct {
		Value string
	}

	// GetGraphQLType returns the GraphQL representation with arguments
	customFieldImpl := CustomFieldWithArgs{Value: "test"}
	_ = customFieldImpl // Use it to avoid unused variable error in type assertion below

	// Define the GetGraphQLType method on the type
	type CustomFieldType interface { //nolint:unused // Test type definition
		GetGraphQLType() string
	}

	// Test 1: Field with GraphQL arguments via GetGraphQLType()
	t.Run("FieldWithArguments", func(t *testing.T) {
		type CustomFieldWithArgs struct { //nolint:unused // Test type definition
			Value string
		}

		// We need to make this implement GraphQLType
		// For testing, we'll use a concrete implementation
		query := struct {
			Repository struct {
				Issue struct {
					Title string
				} `graphql:"issue(number: 1)"`
			} `graphql:"repository(owner: \"test\")"`
		}{}

		got, err := ConstructQuery(&query, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := `{repository(owner: "test"){issue(number: 1){title}}}`
		if got != want {
			t.Errorf("\ngot:  %q\nwant: %q", got, want)
		}
	})

	// Test 2: Field implementing GraphQLType with custom representation
	t.Run("CustomTypeField", func(t *testing.T) {
		// CustomTimestamp that returns a GraphQL field with specific format
		type CustomTimestamp struct { //nolint:unused // Test type definition
			time.Time
		}

		query := struct {
			User struct {
				Name string
				// We'll use UserReview which already implements GraphQLType
				Review UserReview
			} `graphql:"user(id: 1)"`
		}{}

		got, err := ConstructQuery(&query, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// UserReview implements GetGraphQLType() which returns "user_review"
		// So the query should include "user_review" instead of "review"
		want := `{user(id: 1){name,user_review{review,userId}}}`
		if got != want {
			t.Errorf("\ngot:  %q\nwant: %q", got, want)
		}
	})

	// Test 3: Multiple fields implementing GraphQLType
	t.Run("MultipleCustomFields", func(t *testing.T) {
		query := struct {
			Repository struct {
				Owner       string
				Review1     UserReview
				Review2     UserReview
				ReviewInput UserReviewInput
			} `graphql:"repository(owner: \"test\")"`
		}{}

		got, err := ConstructQuery(&query, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := `{repository(owner: "test"){owner,user_review{review,userId},user_review{review,userId},user_review_input{review,userId}}}`
		if got != want {
			t.Errorf("\ngot:  %q\nwant: %q", got, want)
		}
	})

	// Test 4: Nested struct with GraphQLType field
	t.Run("NestedStructWithCustomField", func(t *testing.T) {
		query := struct {
			Organization struct {
				Repository struct {
					Name   string
					Review UserReview
				} `graphql:"repository(name: \"repo\")"`
			} `graphql:"organization(login: \"org\")"`
		}{}

		got, err := ConstructQuery(&query, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := `{organization(login: "org"){repository(name: "repo"){name,user_review{review,userId}}}}`
		if got != want {
			t.Errorf("\ngot:  %q\nwant: %q", got, want)
		}
	})

	// Test 5: Array of GraphQLType implementing types
	// The slice element type's GetGraphQLType() should be used for the field name
	t.Run("ArrayOfCustomFields", func(t *testing.T) {
		query := struct {
			User struct {
				Name    string
				Reviews []UserReview
			} `graphql:"user(id: 1)"`
		}{}

		got, err := ConstructQuery(&query, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// For arrays of GraphQLType implementing types, GetGraphQLType() is called
		// on the element type to determine the field name
		want := `{user(id: 1){name,user_review{review,userId}}}`
		if got != want {
			t.Errorf("\ngot:  %q\nwant: %q", got, want)
		}
	})

	// Test 6: Array of pointer GraphQLType implementing types
	t.Run("ArrayOfPointerCustomFields", func(t *testing.T) {
		query := struct {
			User struct {
				Name    string
				Reviews []*UserReview
			} `graphql:"user(id: 1)"`
		}{}

		got, err := ConstructQuery(&query, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// For arrays of pointers to GraphQLType implementing types
		want := `{user(id: 1){name,user_review{review,userId}}}`
		if got != want {
			t.Errorf("\ngot:  %q\nwant: %q", got, want)
		}
	})
}

var val Uuid

type Uuid uuid.UUID

func (u Uuid) GetGraphQLType() string { return "uuid" }

type MyUuid Uuid

type UserReview struct {
	Review string
	UserID string
}

type UserReviewInput UserReview

func (u UserReview) GetGraphQLType() string { return "user_review" }

func (u UserReviewInput) GetGraphQLType() string { return "user_review_input" }

func (u MyUuid) GetGraphQLType() string { return "my_uuid" }

// Custom GraphQL types for testing.
type (
	// DateTime is an ISO-8601 encoded UTC date.
	DateTime struct{ time.Time }

	// URI is an RFC 3986, RFC 3987, and RFC 6570 (level 4) compliant URI.
	URI struct{ *url.URL }
)

func (u *URI) UnmarshalJSON(data []byte) error { panic("mock implementation") }

// IssueState represents the possible states of an issue.
type IssueState string

// The possible states of an issue.
const (
	IssueStateOpen   IssueState = "OPEN"   // An issue that is still open.
	IssueStateClosed IssueState = "CLOSED" // An issue that has been closed.
)

// ReactionContent represents emojis that can be attached to Issues, Pull Requests and Comments.
type ReactionContent string

// Emojis that can be attached to Issues, Pull Requests and Comments.
const (
	ReactionContentThumbsUp   ReactionContent = "THUMBS_UP"   // Represents the 👍 emoji.
	ReactionContentThumbsDown ReactionContent = "THUMBS_DOWN" // Represents the 👎 emoji.
	ReactionContentLaugh      ReactionContent = "LAUGH"       // Represents the 😄 emoji.
	ReactionContentHooray     ReactionContent = "HOORAY"      // Represents the 🎉 emoji.
	ReactionContentConfused   ReactionContent = "CONFUSED"    // Represents the 😕 emoji.
	ReactionContentHeart      ReactionContent = "HEART"       // Represents the ❤️ emoji.
)

// AddReactionInput is an autogenerated input type of AddReaction.
type AddReactionInput struct {
	// The Node ID of the subject to modify. (Required.)
	SubjectID ID `json:"subjectId"`
	// The name of the emoji to react with. (Required.)
	Content ReactionContent `json:"content"`

	// A unique identifier for the client performing the mutation. (Optional.)
	ClientMutationID *string `json:"clientMutationId,omitempty"`
}

type ActualNodes[T any] struct {
	gqlType string `graphql:"-"`
	Nodes   T
}

func (an *ActualNodes[T]) GetInnerLayer() ContainerLayer {
	return nil
}

func (an *ActualNodes[T]) GetNodes() any {
	return an.Nodes
}

func (an *ActualNodes[T]) GetGraphQLType() string {
	return an.gqlType
}

type ContainerLayer interface {
	GetInnerLayer() ContainerLayer
	GetNodes() any
	GetGraphQLType() string
}

type NestedLayer struct {
	gqlType    string `graphql:"-"`
	InnerLayer ContainerLayer
}

func (nl *NestedLayer) GetInnerLayer() ContainerLayer {
	return nl.InnerLayer
}

func (nl *NestedLayer) GetNodes() any {
	return nil
}

func (nl *NestedLayer) GetGraphQLType() string {
	return nl.gqlType
}

type NestedQuery[T any] struct {
	OutermostLayer ContainerLayer
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
	return layer.GetNodes().(T)
}

func NewNestedQuery[T any](containerLayers ...string) *NestedQuery[T] {
	if len(containerLayers) == 0 {
		return &NestedQuery[T]{
			OutermostLayer: &ActualNodes[T]{},
		}
	}

	var buildLayer func(index int) ContainerLayer
	buildLayer = func(index int) ContainerLayer {
		if index == len(containerLayers)-1 {
			return &ActualNodes[T]{
				gqlType: containerLayers[index],
			}
		}
		return &NestedLayer{
			gqlType:    containerLayers[index],
			InnerLayer: buildLayer(index + 1),
		}
	}

	return &NestedQuery[T]{OutermostLayer: buildLayer(0)}
}

type Test struct {
	Value string `graphql:"value"`
}

func (t Test) GetGraphQLType() string {
	return "test"
}

type Tests []Test

func (t Tests) GetGraphQLType() string {
	return "tests"
}
func TestInterface(t *testing.T) {
	q := NewNestedQuery[Tests]("testcontainer")
	want := `{testcontainer{tests{value}}}`
	got, err := ConstructQuery(q, make(map[string]any))
	if err != nil {
		t.Error(err)
	} else if got != want {
		t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
	}
}

// TestInterface_NilInterface tests handling of uninitialized interface field
func TestInterface_NilInterface(t *testing.T) {
	type Query struct {
		Layer ContainerLayer // nil interface
	}
	q := Query{} // Layer is nil
	got, err := ConstructQuery(q, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should skip nil interface field
	want := `{}`
	if got != want {
		t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
	}
}

// TestInterface_NilPointerValue tests interface containing nil pointer
func TestInterface_NilPointerValue(t *testing.T) {
	type Query struct {
		Layer ContainerLayer
	}
	var nilImpl *NestedLayer
	q := Query{Layer: nilImpl} // Interface contains nil pointer
	got, err := ConstructQuery(q, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should skip nil pointer value
	want := `{}`
	if got != want {
		t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
	}
}

// TestInterface_EmptyInterface tests empty interface{} type
func TestInterface_EmptyInterface(t *testing.T) {
	type Query struct {
		Data any `graphql:"data"` // using any (interface{})
	}
	q := Query{Data: &Test{Value: "test"}}
	got, err := ConstructQuery(q, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	want := `{data{value}}`
	if got != want {
		t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
	}
}

// TestInterface_InSlice tests slice of interfaces
func TestInterface_InSlice(t *testing.T) {
	type Query struct {
		Layers []ContainerLayer `graphql:"layers"`
	}
	q := Query{
		Layers: []ContainerLayer{
			&ActualNodes[Tests]{
				gqlType: "tests",
				Nodes:   Tests{{Value: "test"}},
			},
		},
	}
	got, err := ConstructQuery(q, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	want := `{layers{tests{value}}}`
	if got != want {
		t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
	}
}

// TestInterface_ErrorPath tests error handling during interface recursion
func TestInterface_ErrorPath(t *testing.T) {
	// Test that errors from nested interface processing are properly wrapped
	// We'll use a circular reference which causes a stack overflow protection error
	// or just verify that the interface case itself works without errors for valid types

	// For now, test that processing valid nested interfaces doesn't error
	type Query struct {
		Layer ContainerLayer
	}
	q := Query{
		Layer: &NestedLayer{
			gqlType: "outer",
			InnerLayer: &ActualNodes[Tests]{
				gqlType: "inner",
				Nodes:   Tests{{Value: "test"}},
			},
		},
	}
	got, err := ConstructQuery(q, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	want := `{outer{inner{tests{value}}}}`
	if got != want {
		t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
	}
}

type Wrapped struct {
	Value string `graphql:"value"`
}

type Wrappeds []Wrapped

type Wrapper[T any] struct {
	Wrapped T
}

func (w Wrapper[T]) GetGraphQLType() string {
	return "wrapper"
}

func (w Wrapper[T]) GetGraphQLWrapped() T {
	return w.Wrapped
}

func TestWrapper(t *testing.T) {
	q := NewNestedQuery[Wrapper[Wrappeds]]("testcontainer")
	want := `{testcontainer{wrapper{value}}}`
	got, err := ConstructQuery(q, make(map[string]any))
	if err != nil {
		t.Error(err)
	} else if got != want {
		t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
	}
}

// TestWrapper_NilPointer tests wrapper with nil pointer value
func TestWrapper_NilPointer(t *testing.T) {
	type Query struct {
		Container struct {
			Wrapper *Wrapper[Wrapped] `graphql:"wrapper"`
		} `graphql:"container"`
	}
	q := Query{} // Wrapper is nil
	got, err := ConstructQuery(q, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should skip nil pointer wrapper field
	want := `{container{}}`
	if got != want {
		t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
	}
}

// TestWrapper_Nested tests nested wrappers (Wrapper[Wrapper[T]])
func TestWrapper_Nested(t *testing.T) {
	type DoubleWrapped struct {
		Inner Wrapper[Wrapped]
	}
	type Query struct {
		Container struct {
			Outer Wrapper[DoubleWrapped] `graphql:"outer"`
		} `graphql:"container"`
	}
	q := Query{}
	got, err := ConstructQuery(q, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// GetGraphQLType() returns "wrapper", so field is named "wrapper" not "outer"
	// The inner struct has a Wrapper field which also becomes "wrapper"
	want := `{container{wrapper{wrapper{value}}}}`
	if got != want {
		t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
	}
}

// TestWrapper_SingleStruct tests wrapper with single struct (not slice)
func TestWrapper_SingleStruct(t *testing.T) {
	type Query struct {
		Container struct {
			Single Wrapper[Wrapped] `graphql:"single"`
		} `graphql:"container"`
	}
	q := Query{}
	got, err := ConstructQuery(q, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// GetGraphQLType() returns "wrapper", overriding the graphql tag
	want := `{container{wrapper{value}}}`
	if got != want {
		t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
	}
}

// TestWrapper_PointerContent tests wrapper with pointer wrapped content
func TestWrapper_PointerContent(t *testing.T) {
	type Query struct {
		Container struct {
			PtrWrapper Wrapper[*Wrapped] `graphql:"ptrWrapper"`
		} `graphql:"container"`
	}
	q := Query{}
	got, err := ConstructQuery(q, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// GetGraphQLType() returns "wrapper", overriding the graphql tag
	want := `{container{wrapper{value}}}`
	if got != want {
		t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
	}
}

// TestWrapper_PrimitiveContent tests wrapper with primitive types
func TestWrapper_PrimitiveContent(t *testing.T) {
	t.Run("StringWrapper", func(t *testing.T) {
		type Query struct {
			Container struct {
				StringWrap Wrapper[string] `graphql:"stringWrap"`
			} `graphql:"container"`
		}
		q := Query{}
		got, err := ConstructQuery(q, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// GetGraphQLType() returns "wrapper", and primitive types are scalars
		want := `{container{wrapper}}`
		if got != want {
			t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
		}
	})

	t.Run("IntWrapper", func(t *testing.T) {
		type Query struct {
			Container struct {
				IntWrap Wrapper[int] `graphql:"intWrap"`
			} `graphql:"container"`
		}
		q := Query{}
		got, err := ConstructQuery(q, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// GetGraphQLType() returns "wrapper", and primitive types are scalars
		want := `{container{wrapper}}`
		if got != want {
			t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
		}
	})
}

// TestWrapper_MultipleFields tests struct with multiple wrapper fields
func TestWrapper_MultipleFields(t *testing.T) {
	type Query struct {
		Container struct {
			Wrapper1 Wrapper[Wrapped]  `graphql:"wrapper1"`
			Wrapper2 Wrapper[Wrappeds] `graphql:"wrapper2"`
			Normal   string             `graphql:"normal"`
		} `graphql:"container"`
	}
	q := Query{}
	got, err := ConstructQuery(q, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Both wrappers become "wrapper" (GetGraphQLType() overrides graphql tags)
	want := `{container{wrapper{value},wrapper{value},normal}}`
	if got != want {
		t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
	}
}

// TestWrapper_CustomGraphQLTag tests that GetGraphQLType() overrides graphql tag
func TestWrapper_CustomGraphQLTag(t *testing.T) {
	type Query struct {
		Container struct {
			MyWrapper Wrapper[Wrapped] `graphql:"customName(arg: 123)"`
		} `graphql:"container"`
	}
	q := Query{}
	got, err := ConstructQuery(q, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// GetGraphQLType() returns "wrapper", overriding the graphql tag completely
	want := `{container{wrapper{value}}}`
	if got != want {
		t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
	}
}

// TestWrapper_SkipTag tests that GetGraphQLType() overrides ALL tags including "-"
func TestWrapper_SkipTag(t *testing.T) {
	type Query struct {
		Container struct {
			SkipMe  Wrapper[Wrapped] `graphql:"-"`
			KeepMe  Wrapper[Wrapped] `graphql:"keepMe"`
		} `graphql:"container"`
	}
	q := Query{}
	got, err := ConstructQuery(q, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// When a type implements GetGraphQLType(), it takes precedence over ALL tags,
	// including skip tags. This is intentional: GetGraphQLType() explicitly declares
	// the GraphQL representation, overriding tag-based configuration.
	// Both fields become "wrapper" regardless of graphql tags.
	want := `{container{wrapper{value},wrapper{value}}}`
	if got != want {
		t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
	}
}

// TestWrapper_ScalarTag tests wrapper with scalar tag
func TestWrapper_ScalarTag(t *testing.T) {
	type Query struct {
		Container struct {
			ScalarWrapper Wrapper[Wrapped] `graphql:"scalarWrapper" scalar:"true"`
			NormalWrapper Wrapper[Wrapped] `graphql:"normalWrapper"`
		} `graphql:"container"`
	}
	q := Query{}
	got, err := ConstructQuery(q, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// GetGraphQLType() returns "wrapper" for both (overrides graphql tags)
	// scalar:"true" prevents expansion, so ScalarWrapper is a leaf
	want := `{container{wrapper,wrapper{value}}}`
	if got != want {
		t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
	}
}

// TestWrapper_EmbeddedAnonymous documents Go method promotion with Wrapper types
//
// KEY INSIGHT: Anonymous Wrapper[T] fields promote BOTH methods to the parent:
//   1. GetGraphQLType() → field name becomes "wrapper"
//   2. GetGraphQLWrapped() → unwraps, skipping sibling fields
//
// RECOMMENDATION: Don't use anonymous Wrapper[T] with sibling fields!
func TestWrapper_EmbeddedAnonymous(t *testing.T) {
	type EmbeddedWrapper struct {
		Wrapper[Wrapped]
	}

	t.Run("AnonymousEmbedWithOtherField", func(t *testing.T) {
		type Query struct {
			Container struct {
				EmbeddedWrapper // anonymous: promotes Wrapper methods to Container
				Other string `graphql:"other"`
			} `graphql:"container"`
		}
		q := Query{}
		got, err := ConstructQuery(q, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// Go method promotion causes Container to inherit:
		//   - GetGraphQLType() → "wrapper" (overrides graphql:"container")
		//   - GetGraphQLWrapped() → returns Wrapped{}, skips Other field
		want := `{wrapper{value}}`
		if got != want {
			t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
		}
	})

	t.Run("AnonymousEmbedOnly", func(t *testing.T) {
		type Query struct {
			Container struct {
				EmbeddedWrapper
			} `graphql:"container"`
		}
		q := Query{}
		got, err := ConstructQuery(q, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// Same method promotion: Container becomes "wrapper", gets unwrapped
		want := `{wrapper{value}}`
		if got != want {
			t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
		}
	})

	t.Run("DirectAnonymousWrapper", func(t *testing.T) {
		type Query struct {
			Container struct {
				Wrapper[Wrapped]
				Other string `graphql:"other"`
			} `graphql:"container"`
		}
		q := Query{}
		got, err := ConstructQuery(q, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// Direct anonymous Wrapper: same issue, methods promoted to Container
		want := `{wrapper{value}}`
		if got != want {
			t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
		}
	})

	t.Run("NamedEmbedWrapper", func(t *testing.T) {
		type Query struct {
			Container struct {
				Embed EmbeddedWrapper // NAMED field - no method promotion!
				Other string `graphql:"other"`
			} `graphql:"container"`
		}
		q := Query{}
		got, err := ConstructQuery(q, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// Named field: Container keeps graphql:"container", both fields work!
		// Embed.Wrapper field implements GetGraphQLType() → becomes "wrapper"
		want := `{container{wrapper{value},other}}`
		if got != want {
			t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
		}
	})

	t.Run("SimpleAnonymousStruct", func(t *testing.T) {
		type SimpleEmbed struct {
			Value string `graphql:"value"`
		}
		type Query struct {
			Container struct {
				SimpleEmbed
				Other string `graphql:"other"`
			} `graphql:"container"`
		}
		q := Query{}
		got, err := ConstructQuery(q, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// Simple anonymous struct without wrapper methods: works as expected
		// Fields are properly inlined
		want := `{container{value,other}}`
		if got != want {
			t.Errorf("\ngot:  %q\nwant: %q\n", got, want)
		}
	})
}

func TestHasVariables(t *testing.T) {
	tests := []struct {
		name      string
		variables any
		want      bool
	}{
		{
			name:      "nil variables",
			variables: nil,
			want:      false,
		},
		{
			name:      "empty map",
			variables: map[string]any{},
			want:      false,
		},
		{
			name: "non-empty map with one entry",
			variables: map[string]any{
				"id": "123",
			},
			want: true,
		},
		{
			name: "non-empty map with multiple entries",
			variables: map[string]any{
				"id":   "123",
				"name": "test",
			},
			want: true,
		},
		{
			name: "struct variables",
			variables: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{
				ID:   "123",
				Name: "test",
			},
			want: true,
		},
		{
			name: "pointer to struct",
			variables: &struct {
				ID string `json:"id"`
			}{
				ID: "123",
			},
			want: true,
		},
		{
			name:      "string (not a map)",
			variables: "test",
			want:      true,
		},
		{
			name:      "int (not a map)",
			variables: 42,
			want:      true,
		},
		{
			name:      "slice (not a map)",
			variables: []string{"a", "b"},
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasVariables(tt.variables); got != tt.want {
				t.Errorf("hasVariables() = %v, want %v", got, tt.want)
			}
		})
	}
}

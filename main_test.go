package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/kinbiko/jsonassert"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

const simultaneousUsers = 500
const numItems = 99

type TestCase struct {
	Username string
	Index    int
	Bid      int
}

var app *App

func TestMain(m *testing.M) {
	app = new(App)
	app.init()
	m.Run()
}

func TestListAllItems(t *testing.T) {
	jsn := jsonassert.New(t)
	context, recorder := get("/items")
	if assert.NoError(t, app.ListAllItems(context)) {
		assert.Equal(t, http.StatusOK, recorder.Code)

		var items []map[string]interface{}
		json.Unmarshal([]byte(recorder.Body.String()), &items)

		assert.Equal(t, numItems, len(items))

		for index, item := range items {
			itemJSON, _ := json.Marshal(item)
			jsn.Assertf(string(itemJSON), `{
				"id": %d,
				"name": "<<PRESENCE>>",
				"description": "<<PRESENCE>>"
			}`, index+1)
		}
	}
}

func TestGetOneItem(t *testing.T) {
	jsn := jsonassert.New(t)
	i := rand.Intn(numItems-1) + 1
	istr := strconv.Itoa(i)

	context, recorder := get("/items/:id")
	context.SetParamNames("id")
	context.SetParamValues(istr)

	if assert.NoError(t, app.GetOneItem(context)) {
		assert.Equal(t, http.StatusOK, recorder.Code)

		jsn.Assertf(recorder.Body.String(), `{
			"id": %d,
			"name": "<<PRESENCE>>",
			"description": "<<PRESENCE>>"
		}`, i)
	}
}

// Bidding is the core of the application and the most sensitive
// to race conditions, because multiple endpoints share the same
// memory space to read and write bids.
//
// This test simulates an auction with thousands of simultaneous
// bidders. Users will concurrently create a random Bid for all
// Items. They will always bid the same value, so in the end we
// should expect a single user winning all items.
//
// Due to the shared state condition between multiple endpoints,
// we will run a sequence of requests in different endpoints to
// test how one call affects the next.

func TestBidConcurrency(t *testing.T) {
	winner := 0
	highestBid := 0

	//This function will block until all subtests are finished
	t.Run("TestGroup", func(t *testing.T) {
		testCases := []TestCase{}
		for i := 0; i < simultaneousUsers; i++ {
			bid := rand.Intn(1000000)
			if bid > highestBid {
				highestBid = bid
				winner = i
			}

			testCases = append(testCases, TestCase{
				Index:    i,
				Username: "user" + strconv.Itoa(i),
				Bid:      bid,
			})
		}

		for _, tc := range testCases {
			tc := tc

			// Flood the API with simultaneous requests
			t.Run("TestCase:"+tc.Username, func(t *testing.T) {
				tc := tc
				t.Parallel()
				jsn := jsonassert.New(t)
				var jwtToken *jwt.Token

				// Login and get a JWT Token
				t.Run("login:"+tc.Username, func(t *testing.T) {
					form := make(url.Values)
					form.Set("username", tc.Username)
					context, recorder := post("/login", form.Encode())

					if assert.NoError(t, Login(context)) {
						assert.Equal(t, http.StatusOK, recorder.Code)

						token, claims := parseToken(recorder.Body.String())
						username := claims["username"].(string)
						assert.Equal(t, tc.Username, username)

						jwtToken = token
					}
				})

				t.Run("create_bids", func(t *testing.T) {
					tc := tc
					for itemID := 1; itemID <= numItems; itemID++ {
						tc := tc
						itemID := itemID
						t.Run("bid:"+strconv.Itoa(itemID), func(t *testing.T) {
							t.Parallel()
							form := make(url.Values)
							form.Set("value", strconv.Itoa(tc.Bid))
							context, recorder := signedPost("/items/:id/bids/create", form.Encode(), jwtToken.Raw)
							context.SetParamNames("id")
							context.SetParamValues(strconv.Itoa(itemID))

							if assert.NoError(t, app.CreateBid(context)) {
								assert.Equal(t, http.StatusCreated, recorder.Code)

								jsn.Assertf(recorder.Body.String(), `{
									"username": "%s",
									"value": %d,
									"created_at": "<<PRESENCE>>",
									"item_id": %d,
									"item_name": "<<PRESENCE>>"
								}`, tc.Username, tc.Bid, itemID)
							}
						})
					}
				})

				t.Run("view_bids:", func(t *testing.T) {
					context, recorder := signedGet("/items/my_bids", jwtToken.Raw)
					if assert.NoError(t, app.ListUserBidItems(context)) {
						assert.Equal(t, http.StatusOK, recorder.Code)

						var items []map[string]interface{}
						json.Unmarshal([]byte(recorder.Body.String()), &items)

						assert.Equal(t, numItems, len(items))

						itemJSON, _ := json.Marshal(items[numItems-1])
						jsn.Assertf(string(itemJSON), `{
							"id": "<<PRESENCE>>",
							"name": "<<PRESENCE>>",
							"description": "<<PRESENCE>>"
						}`)
					}
				})
			})
		}
	})

	jsn := jsonassert.New(t)
	for i := 1; i <= numItems; i++ {
		context, recorder := get("/items/:id/bids")
		context.SetParamNames("id")
		context.SetParamValues(strconv.Itoa(i))

		if assert.NoError(t, app.GetBidsOnItem(context)) {
			assert.Equal(t, http.StatusOK, recorder.Code)

			var bids []map[string]interface{}
			json.Unmarshal([]byte(recorder.Body.String()), &bids)

			// Winner should be ranked first
			bidJSON, _ := json.Marshal(bids[0])
			jsn.Assertf(string(bidJSON), `{
				"username": "user%d",
				"value": %d,
				"created_at": "<<PRESENCE>>",
				"item_id": "<<PRESENCE>>",
				"item_name": "<<PRESENCE>>"
			}`, winner, highestBid)
		}

		context, recorder = get("/items/:id/bids/winning")
		context.SetParamNames("id")
		context.SetParamValues(strconv.Itoa(i))

		if assert.NoError(t, app.GetWinningBid(context)) {
			assert.Equal(t, http.StatusOK, recorder.Code)

			jsn.Assertf(recorder.Body.String(), `{
				"username": "user%d",
				"value": %d,
				"created_at": "<<PRESENCE>>",
				"item_id": %d,
				"item_name": "<<PRESENCE>>"
			}`, winner, highestBid, i)
		}
	}
}

// Helper functions

func get(url string) (echo.Context, *httptest.ResponseRecorder) {
	return signedGet(url, "")
}

func post(url string, body string) (echo.Context, *httptest.ResponseRecorder) {
	return signedPost(url, body, "")
}

func signedGet(url string, token string) (echo.Context, *httptest.ResponseRecorder) {
	r := httptest.NewRequest(http.MethodGet, url, nil)
	return request(r, token)
}

func signedPost(url string, body string, token string) (echo.Context, *httptest.ResponseRecorder) {
	r := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
	r.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	return request(r, token)
}

func request(request *http.Request, tokenString string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()

	// Workaround for https://github.com/labstack/echo/issues/1492
	e.GET("/:param", func(e echo.Context) error { return nil })
	// @TODO remove above line after issue is fixed

	recorder := httptest.NewRecorder()
	context := e.NewContext(request, recorder)

	if tokenString != "" {
		request.Header.Set("Authorization", "Bearer "+tokenString)
		token, _ := jwt.Parse(tokenString, nil)
		context.Set("user", token)
	}

	return context, recorder
}

func parseToken(body string) (*jwt.Token, jwt.MapClaims) {
	var content map[string]string
	json.Unmarshal([]byte(body), &content)
	token, _ := jwt.Parse(content["token"], nil)
	claims := token.Claims.(jwt.MapClaims)
	return token, claims
}

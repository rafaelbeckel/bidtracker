package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// App holds a collection of users and items
type App struct {
	Users    map[string]*User
	Items    map[int]*Item
	ItemList []*Item
	mutex    sync.Mutex
}

func (a *App) init() {
	a.Users = make(map[string]*User)
	a.Items = make(map[int]*Item)
	a.createItemsDB()
}

func (a *App) createItemsDB() {
	file, err := os.Open("items.json")
	defer file.Close()
	if err != nil {
		log.Println("Could not open items.json")
		os.Exit(1)
	}

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println("Could not read items.json")
		os.Exit(2)
	}

	var items []map[string]interface{}
	err = json.Unmarshal([]byte(bytes), &items)
	if err != nil {
		log.Println("Malformed Json in items.json")
		os.Exit(3)
	}

	a.ItemList = make([]*Item, len(items))
	for index, item := range items {
		id := int(item["id"].(float64))
		a.Items[id] = &Item{
			ID:          id,
			Name:        item["name"].(string),
			Description: item["description"].(string),
		}
		a.Items[id].Init()
		a.ItemList[index] = a.Items[id]
	}
}

func (a *App) getOrCreateUser(token *jwt.Token) *User {
	claims := token.Claims.(jwt.MapClaims)
	username := claims["username"].(string)

	if _, ok := a.Users[username]; !ok {
		a.Users[username] = &User{
			Username: username,
			Bids:     []*Bid{},
		}
	}
	return a.Users[username]
}

// ListAllItems <- GET /items
func (a *App) ListAllItems(c echo.Context) error {
	return c.JSON(http.StatusOK, a.ItemList)
}

// GetOneItem <- GET /items/:id
func (a *App) GetOneItem(c echo.Context) error {
	itemID, _ := strconv.Atoi(c.Param("id"))
	if item, ok := a.Items[itemID]; ok {
		return c.JSON(http.StatusOK, item)
	}
	return c.String(http.StatusNotFound, `{"message":"Item not found!"}`)
}

// GetBidsOnItem <- GET /items/:id/bids
func (a *App) GetBidsOnItem(c echo.Context) error {
	itemID, _ := strconv.Atoi(c.Param("id"))
	if item, ok := a.Items[itemID]; ok {
		return c.JSON(http.StatusOK, item.GetAllBids())
	}
	return c.String(http.StatusNotFound, `{"message":"Item not found!"}`)
}

// GetWinningBid <- /items/:id/bids/winning
func (a *App) GetWinningBid(c echo.Context) error {
	itemID, _ := strconv.Atoi(c.Param("id"))
	if item, ok := a.Items[itemID]; ok {
		if item.HasBids() {
			return c.JSON(http.StatusOK, item.GetWinningBid())
		}
		return c.String(http.StatusNoContent, `{"message":"Item has not received bids"}`)
	}
	return c.String(http.StatusNotFound, `{"message":"Item not found!"}`)
}

// ListUserBidItems <- GET /items/my_bids
func (a *App) ListUserBidItems(c echo.Context) error {
	token := c.Get("user").(*jwt.Token)
	u := a.getOrCreateUser(token)
	return c.JSON(http.StatusOK, u.GetBidItems())
}

// CreateBid <- POST /items/:id/bids/create
func (a *App) CreateBid(c echo.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	itemID, _ := strconv.Atoi(c.Param("id"))
	value, _ := strconv.Atoi(c.FormValue("value"))
	token := c.Get("user").(*jwt.Token)
	user := a.getOrCreateUser(token)

	if item, ok := a.Items[itemID]; ok && value > 0 {
		bid := user.CreateBid(item, value)
		item.RecordBid(bid)
		return c.JSON(http.StatusCreated, bid)
	}
	return c.String(http.StatusNotFound, `{"message":"Item not found!"}`)
}

func main() {
	app := new(App)
	app.init()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	loggedIn := middleware.JWTWithConfig(middleware.JWTConfig{
		SigningKey: []byte(JWTSecret),
	})

	// Public routes
	e.POST("/login", Login)
	e.GET("/items", app.ListAllItems)
	e.GET("/items/:id", app.GetOneItem)
	e.GET("/items/:id/bids", app.GetBidsOnItem)
	e.GET("/items/:id/bids/winning", app.GetWinningBid)

	// Protected routes
	e.GET("/items/my_bids", app.ListUserBidItems, loggedIn)
	e.POST("/items/:id/bids/create", app.CreateBid, loggedIn)

	e.Logger.Fatal(e.Start(":3000"))
}

package main

import (
	"sync"
	"time"

	"github.com/google/btree"
)

// Bid implements btree.Item interface for easy value comparison
type Bid struct {
	Username  string    `json:"username"`
	Value     int       `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	ItemID    int       `json:"item_id"`
	ItemName  string    `json:"item_name"`
	item      *Item
	accepted  bool
}

// User represents a person or device who interacts with
// our API to post bids on items and read some item data
type User struct {
	Username string `json:"username"`
	Bids     []*Bid `json:"bids"`
	mutex    sync.Mutex
}

// Less returns true if the provided bid > current bid
func (b Bid) Less(a btree.Item) bool {
	return b.Value < a.(Bid).Value
}

// GetBidItems gets all the items on which a user has given a bid
func (u *User) GetBidItems() []*Item {
	items := []*Item{}
	for _, bid := range u.Bids {
		items = append(items, bid.item)
	}
	return items
}

// CreateBid creates a new Bid on a Item
func (u *User) CreateBid(item *Item, value int) Bid {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	bid := Bid{
		Username:  u.Username,
		Value:     value,
		CreatedAt: time.Now(),
		ItemID:    item.ID,
		ItemName:  item.Name,
		accepted:  false,
		item:      item,
	}

	u.Bids = append(u.Bids, &bid)

	return bid
}

package main

import (
	"flag"
	"sync"

	"github.com/google/btree"
)

// Item stores metadata and a list of ranked user's bids.
// It enforces thread-safe insert operations on the list.
type Item struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	bids        *btree.BTree
	mutex       sync.Mutex
}

var btreeDegree = flag.Int("degree", 2, "B-Tree degree")

// Init creates an empty internal representation of Bids
func (i *Item) Init() {
	i.bids = btree.New(*btreeDegree)
}

// RecordBid saves a given bid if it's higher than the max bid
func (i *Item) RecordBid(bid Bid) {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	var max = i.bids.Max()
	if max == nil || max.Less(bid) {
		bid.accepted = true
		i.bids.ReplaceOrInsert(bid)
	}
}

// GetWinningBid returns the highest bid for this item
func (i *Item) GetWinningBid() Bid {
	if i.HasBids() {
		return i.bids.Max().(Bid)
	}
	return Bid{}
}

// HasBids returns true if the item has at least one bid
func (i *Item) HasBids() bool {
	return i.bids.Len() > 0
}

// GetAllBids lists all bids given for this item
func (i *Item) GetAllBids() []Bid {
	bids := []Bid{}
	i.bids.Descend(func(bid btree.Item) bool {
		bids = append(bids, bid.(Bid))
		return true
	})

	return bids
}

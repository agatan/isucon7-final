package main

import (
	"sync"

	"github.com/jmoiron/sqlx"
)

var MasterItems map[int]*mItem
var priceByCountByItemID *sync.Map
var powerByCountByItemID *sync.Map

func initMasterItems(db *sqlx.DB) {
	mItems := map[int]*mItem{}
	var items []*mItem
	err := db.Select(&items, "SELECT * FROM m_item")
	if err != nil {
		panic(err)
	}
	for _, item := range items {
		mItems[item.ItemID] = item
	}
	MasterItems = mItems
}

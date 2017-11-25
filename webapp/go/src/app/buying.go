package main

import (
	"fmt"

	"github.com/izumin5210/ro"
	"github.com/izumin5210/ro/types"
)

var (
	buyingStore ro.Store
)

type Buying struct {
	ro.Model
	RoomName string `db:"room_name"`
	ItemID   int    `db:"item_id"`
	Ordinal  int    `db:"ordinal"`
	Time     int64  `db:"time"`
}

func (b *Buying) GetKeySuffix() string {
	return fmt.Sprintf("%s:%d:%d", b.RoomName, b.ItemID, b.Ordinal)
}

var buyingScorerFuncs = []types.ScorerFunc{
	func(m types.Model) (string, interface{}) {
		b := m.(*Buying)
		return fmt.Sprintf("%s:item_id", b.RoomName), b.ItemID
	},
	func(m types.Model) (string, interface{}) {
		b := m.(*Buying)
		return fmt.Sprintf("%s:time", b.RoomName), b.Time
	},
}

func initBuyingStore() {
	var err error
	buyingStore, err = ro.New(redisPool.Get, &Adding{}, ro.WithScorers(buyingScorerFuncs))
	if err != nil {
		panic(err)
	}
}

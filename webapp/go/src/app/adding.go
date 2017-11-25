package main

import (
	"fmt"

	"github.com/izumin5210/ro"
	"github.com/izumin5210/ro/types"
)

var (
	addingStore ro.Store
)

type Adding struct {
	ro.Model
	RoomName string `json:"-" db:"room_name" redis:"-"`
	Time     int64  `json:"time" db:"time" redis:"-"`
	Isu      string `json:"isu" db:"isu" redis:"isu"`
}

func (a *Adding) GetKeySuffix() string {
	return fmt.Sprintf("%s:%d", a.RoomName, a.Time)
}

var addingScorerFuncs = []types.ScorerFunc{
	func(m types.Model) (string, interface{}) {
		a := m.(*Adding)
		return fmt.Sprintf("%s:time", a.RoomName), a.Time
	},
}

func initAddingStore() {
	var err error
	addingStore, err = ro.New(redisPool.Get, &Adding{}, ro.WithScorers(addingScorerFuncs))
	if err != nil {
		panic(err)
	}
}

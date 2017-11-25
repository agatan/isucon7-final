package main

import (
	"github.com/izumin5210/ro"
)

var (
	roomItemStore   ro.Store
	roomStatusStore ro.Store
)

type RoomStatus struct {
	RoomName    string `redis:"-"`
	Exponential `redis:"-"`
}

func (rs *RoomStatus) GetKeySuffix() string {
	return rs.RoomName
}

type RoomItem struct {
	RoomName string `redis:"-"`
	ItemID   int64  `redis:"item_id"`
	Count    int    `redis:"count"`
}

func (ri *RoomItem) GetKeySuffix() string {
	return ri.RoomName
}

func initRoomStatusStore() {
	var err error
	roomItemStore, err = ro.New(redisPool.Get, &RoomStatus{})
	if err != nil {
		panic(err)
	}
}

func initRoomItemStore() {
	var err error
	roomItemStore, err = ro.New(redisPool.Get, &RoomStatus{})
	if err != nil {
		panic(err)
	}
}

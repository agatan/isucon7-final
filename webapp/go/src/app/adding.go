package main

import (
	"log"
	"math/big"
	"sync"
)

var (
	muxByRoomName map[string]*sync.Mutex
)

type Adding struct {
	RoomName string `json:"-" db:"room_name"`
	Time     int64  `json:"time" db:"time"`
	Isu      string `json:"isu" db:"isu"`
}

func initMuxByRoomName() {
	muxByRoomName = map[string]*sync.Mutex{}
}

func addIsu(roomName string, reqIsu *big.Int, reqTime int64) bool {
	mu := muxByRoomName[roomName]
	mu.Lock()
	defer mu.Unlock()

	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return false
	}

	_, ok := updateRoomTime(tx, roomName, reqTime)
	if !ok {
		tx.Rollback()
		return false
	}

	var isuStr string
	err = tx.QueryRow("SELECT isu FROM adding WHERE room_name = ? AND time = ? ", roomName, reqTime).Scan(&isuStr)
	if err != nil {
		log.Println(err)
		tx.Rollback()
		return false
	}
	isu := str2big(isuStr)

	isu.Add(isu, reqIsu)
	_, err = tx.Exec("INSERT INTO adding(room_name, time, isu) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE isu=isu", roomName, reqTime, isu.String())
	if err != nil {
		log.Println(err)
		tx.Rollback()
		return false
	}

	if err := tx.Commit(); err != nil {
		log.Println(err)
		return false
	}
	return true
}

package main

import (
	"log"
	"time"
)

var (
	roomTimeByName map[string]int64
)

func updateRoomTime(roomName string, reqTime int64) (int64, bool) {
	roomTime := roomTimeByName[roomName]

	var currentTime int64 = int64(time.Now().UnixNano()) / 1000000
	if roomTime > currentTime {
		log.Println("room time is future")
		return 0, false
	}
	if reqTime != 0 {
		if reqTime < currentTime {
			log.Println("reqTime is past")
			return 0, false
		}
	}

	roomTimeByName[roomName] = currentTime

	return currentTime, true
}

func initRoomTime() {
	roomTimeByName = map[string]int64{}
}

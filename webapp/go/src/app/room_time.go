package main

import (
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

// 部屋のロックを取りタイムスタンプを更新する
//
// トランザクション開始後この関数を呼ぶ前にクエリを投げると、
// そのトランザクション中の通常のSELECTクエリが返す結果がロック取得前の
// 状態になることに注意 (keyword: MVCC, repeatable read).
func updateRoomTime(tx *sqlx.Tx, roomName string, reqTime int64) (int64, bool) {
	// See page 13 and 17 in https://www.slideshare.net/ichirin2501/insert-51938787
	_, err := tx.Exec("INSERT INTO room_time(room_name, time) VALUES (?, 0) ON DUPLICATE KEY UPDATE time = time", roomName)
	if err != nil {
		log.Println(err)
		return 0, false
	}

	var roomTime int64
	err = tx.Get(&roomTime, "SELECT time FROM room_time WHERE room_name = ? FOR UPDATE", roomName)
	if err != nil {
		log.Println(err)
		return 0, false
	}

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

	_, err = tx.Exec("UPDATE room_time SET time = ? WHERE room_name = ?", currentTime, roomName)
	if err != nil {
		log.Println(err)
		return 0, false
	}

	return currentTime, true
}

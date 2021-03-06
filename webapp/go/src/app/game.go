package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/big"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
)

type GameRequest struct {
	RequestID int    `json:"request_id"`
	Action    string `json:"action"`
	Time      int64  `json:"time"`

	// for addIsu
	Isu string `json:"isu"`

	// for buyItem
	ItemID      int `json:"item_id"`
	CountBought int `json:"count_bought"`
}

type GameResponse struct {
	RequestID int  `json:"request_id"`
	IsSuccess bool `json:"is_success"`
}

// 10進数の指数表記に使うデータ。JSONでは [仮数部, 指数部] という2要素配列になる。
type Exponential struct {
	// Mantissa * 10 ^ Exponent
	Mantissa int64
	Exponent int64
}

func (n Exponential) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("[%d,%d]", n.Mantissa, n.Exponent)), nil
}

type Schedule struct {
	Time       int64       `json:"time"`
	MilliIsu   Exponential `json:"milli_isu"`
	TotalPower Exponential `json:"total_power"`
}

type Item struct {
	ItemID      int         `json:"item_id"`
	CountBought int         `json:"count_bought"`
	CountBuilt  int         `json:"count_built"`
	NextPrice   Exponential `json:"next_price"`
	Power       Exponential `json:"power"`
	Building    []Building  `json:"building"`
}

type OnSale struct {
	ItemID int   `json:"item_id"`
	Time   int64 `json:"time"`
}

type Building struct {
	Time       int64       `json:"time"`
	CountBuilt int         `json:"count_built"`
	Power      Exponential `json:"power"`
}

type GameStatus struct {
	Time     int64      `json:"time"`
	Adding   []*Adding  `json:"adding"`
	Schedule []Schedule `json:"schedule"`
	Items    []Item     `json:"items"`
	OnSale   []OnSale   `json:"on_sale"`
}

type mItem struct {
	ItemID int   `db:"item_id"`
	Power1 int64 `db:"power1"`
	Power2 int64 `db:"power2"`
	Power3 int64 `db:"power3"`
	Power4 int64 `db:"power4"`
	Price1 int64 `db:"price1"`
	Price2 int64 `db:"price2"`
	Price3 int64 `db:"price3"`
	Price4 int64 `db:"price4"`
}

func (item *mItem) GetPower(count int) *big.Int {
	// power(x):=(cx+1)*d^(ax+b)
	a := item.Power1
	b := item.Power2
	c := item.Power3
	d := item.Power4
	x := int64(count)

	s := big.NewInt(c*x + 1)
	t := new(big.Int).Exp(big.NewInt(d), big.NewInt(a*x+b), nil)
	return new(big.Int).Mul(s, t)
}

func (item *mItem) GetPrice(count int) *big.Int {
	// price(x):=(cx+1)*d^(ax+b)
	a := item.Price1
	b := item.Price2
	c := item.Price3
	d := item.Price4
	x := int64(count)

	s := big.NewInt(c*x + 1)
	t := new(big.Int).Exp(big.NewInt(d), big.NewInt(a*x+b), nil)
	return new(big.Int).Mul(s, t)
}

func addIsu(roomName string, reqIsu *big.Int, reqTime int64) bool {
	muxByRoomNameMu.Lock()
	mu := muxByRoomName[roomName]
	muxByRoomNameMu.Unlock()
	mu.Lock()
	defer mu.Unlock()

	_, ok := updateRoomTime(nil, roomName, reqTime)
	if !ok {
		return false
	}

	a := &Adding{RoomName: roomName, Time: reqTime}
	err := addingStore.Get(a)
	if err != nil {
		a.Isu = "0"
	}
	//
	// _, err = tx.Exec("INSERT INTO adding(room_name, time, isu) VALUES (?, ?, '0') ON DUPLICATE KEY UPDATE isu=isu", roomName, reqTime)
	// if err != nil {
	//   log.Println(err)
	//   tx.Rollback()
	//   return false
	// }
	//
	// var isuStr string
	// err = tx.QueryRow("SELECT isu FROM adding WHERE room_name = ? AND time = ? FOR UPDATE", roomName, reqTime).Scan(&isuStr)
	// if err != nil {
	//   log.Println(err)
	//   tx.Rollback()
	//   return false
	// }
	isu := str2big(a.Isu)

	isu.Add(isu, reqIsu)
	a.Isu = isu.String()
	err = addingStore.Set(a)
	if err != nil {
		log.Println(err)
		return false
	}
	// _, err = tx.Exec("UPDATE adding SET isu = ? WHERE room_name = ? AND time = ?", isu.String(), roomName, reqTime)
	// if err != nil {
	//   log.Println(err)
	//   tx.Rollback()
	//   return false
	// }
	//
	// if err := tx.Commit(); err != nil {
	//   log.Println(err)
	//   return false
	// }
	return true
}

func buyItem(roomName string, itemID int, countBought int, reqTime int64) bool {
	muxByRoomNameMu.Lock()
	mu := muxByRoomName[roomName]
	muxByRoomNameMu.Unlock()
	mu.Lock()
	defer mu.Unlock()

	// tx, err := db.Beginx()
	// if err != nil {
	//   log.Println(err)
	//   return false
	// }

	_, ok := updateRoomTime(nil, roomName, reqTime)
	if !ok {
		// tx.Rollback()
		return false
	}

	countBuying, err := buyingStore.Count(buyingStore.Query(fmt.Sprintf("%s:item_id", roomName)).Eq(itemID))
	// var countBuying int
	// err = tx.Get(&countBuying, "SELECT COUNT(*) FROM buying WHERE room_name = ? AND item_id = ?", roomName, itemID)
	if err != nil {
		log.Println(err)
		// tx.Rollback()
		return false
	}
	if countBuying != countBought {
		// tx.Rollback()
		log.Println(roomName, itemID, countBought+1, " is already bought")
		return false
	}

	var addings []*Adding
	err = addingStore.Select(&addings, addingStore.Query(fmt.Sprintf("%s:time", roomName)).LtEq(reqTime))
	// err = tx.Select(&addings, "SELECT isu FROM adding WHERE room_name = ? AND time <= ?", roomName, reqTime)
	if err != nil {
		log.Println(err)
		// tx.Rollback()
		return false
	}

	totalMilliIsu := new(big.Int)
	for _, a := range addings {
		totalMilliIsu.Add(totalMilliIsu, new(big.Int).Mul(str2big(a.Isu), bi1000))
	}

	var buyings []*Buying
	err = buyingStore.Select(&buyings, buyingStore.Query(fmt.Sprintf("%s:time", roomName)))
	// err = tx.Select(&buyings, "SELECT item_id, ordinal, time FROM buying WHERE room_name = ?", roomName)
	if err != nil {
		log.Println(err)
		// tx.Rollback()
		return false
	}
	for _, b := range buyings {
		var item *mItem = MasterItems[b.ItemID]
		cost := new(big.Int).Mul(item.GetPrice(b.Ordinal), bi1000)
		totalMilliIsu.Sub(totalMilliIsu, cost)
		if b.Time <= reqTime {
			gain := new(big.Int).Mul(item.GetPower(b.Ordinal), big.NewInt(reqTime-b.Time))
			totalMilliIsu.Add(totalMilliIsu, gain)
		}
	}

	var item *mItem = MasterItems[itemID]
	need := new(big.Int).Mul(item.GetPrice(countBought+1), bi1000)
	if totalMilliIsu.Cmp(need) < 0 {
		log.Println("not enough")
		// tx.Rollback()
		return false
	}

	err = buyingStore.Set(&Buying{
		RoomName: roomName,
		ItemID:   itemID,
		Ordinal:  countBought + 1,
		Time:     reqTime,
	})
	// _, err = tx.Exec("INSERT INTO buying(room_name, item_id, ordinal, time) VALUES(?, ?, ?, ?)", roomName, itemID, countBought+1, reqTime)
	if err != nil {
		log.Println(err)
		// tx.Rollback()
		return false
	}

	// if err := tx.Commit(); err != nil {
	//   log.Println(err)
	//   return false
	// }

	return true
}

func getStatus(roomName string) (*GameStatus, error) {
	muxByRoomNameMu.Lock()
	mu := muxByRoomName[roomName]
	muxByRoomNameMu.Unlock()
	mu.Lock()
	defer mu.Unlock()

	tx, err := db.Beginx()
	if err != nil {
		return nil, err
	}

	currentTime, ok := updateRoomTime(tx, roomName, 0)
	if !ok {
		tx.Rollback()
		return nil, fmt.Errorf("updateRoomTime failure")
	}

	mItems := MasterItems

	addings := []*Adding{}
	err = addingStore.Select(&addings, addingStore.Query(fmt.Sprintf("%s:time", roomName)))
	// err = tx.Select(&addings, "SELECT time, isu FROM adding WHERE room_name = ?", roomName)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	deletableTimes := []int64{}
	var totalIsu = big.NewInt(0)
	newAddings := []*Adding{{RoomName: roomName, Time: currentTime, Isu: totalIsu.String()}}
	for _, a := range addings {
		// adding は adding.time に isu を増加させる
		if a.Time <= currentTime {
			totalIsu.Add(totalIsu, str2big(a.Isu))
			deletableTimes = append(deletableTimes, a.Time)
		} else {
			newAddings = append(newAddings, a)
		}
	}
	newAddings[0].Isu = totalIsu.String()
	if len(deletableTimes) > 0 {
		addingStore.RemoveBy(addingStore.Query(fmt.Sprintf("%s:time", roomName)).LtEq(currentTime))
		// query, args, err := sqlx.In("DELETE FROM adding WHERE room_name = ? AND time in (?)", roomName, deletableTimes)
		// if err != nil {
		//   tx.Rollback()
		//   return nil, err
		// }
		// _, err = tx.Exec(query, args...)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	err = addingStore.Set(newAddings[0])
	// _, err = tx.Exec("INSERT INTO adding(room_name, time, isu) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE isu=isu", roomName, currentTime, totalIsu.String())
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	buyings := []*Buying{}
	err = buyingStore.Select(&buyings, buyingStore.Query(fmt.Sprintf("%s:time", roomName)))
	// err = tx.Select(&buyings, "SELECT item_id, ordinal, time FROM buying WHERE room_name = ?", roomName)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	status, err := calcStatus(currentTime, mItems, newAddings, buyings)
	if err != nil {
		return nil, err
	}

	// calcStatusに時間がかかる可能性があるので タイムスタンプを取得し直す

	status.Time = int64(time.Now().UnixNano()) / 1000000
	return status, err
}

func calcStatus(currentTime int64, mItems map[int]*mItem, addings []*Adding, buyings []*Buying) (*GameStatus, error) {
	var (
		// 1ミリ秒に生産できる椅子の単位をミリ椅子とする
		totalMilliIsu = big.NewInt(0)
		totalPower    = big.NewInt(0)

		itemPower    = map[int]*big.Int{}    // ItemID => Power
		itemPrice    = map[int]*big.Int{}    // ItemID => Price
		itemOnSale   = map[int]int64{}       // ItemID => OnSale
		itemBuilt    = map[int]int{}         // ItemID => BuiltCount
		itemBought   = map[int]int{}         // ItemID => CountBought
		itemBuilding = map[int][]Building{}  // ItemID => Buildings
		itemPower0   = map[int]Exponential{} // ItemID => currentTime における Power
		itemBuilt0   = map[int]int{}         // ItemID => currentTime における BuiltCount

		addingAt = map[int64]*Adding{}   // Time => currentTime より先の Adding
		buyingAt = map[int64][]*Buying{} // Time => currentTime より先の Buying
	)

	for itemID := range mItems {
		itemPower[itemID] = big.NewInt(0)
		itemBuilding[itemID] = []Building{}
	}

	for _, a := range addings {
		// adding は adding.time に isu を増加させる
		if a.Time <= currentTime {
			totalMilliIsu.Add(totalMilliIsu, new(big.Int).Mul(str2big(a.Isu), bi1000))
		} else {
			addingAt[a.Time] = a
		}
	}

	for _, b := range buyings {
		// buying は 即座に isu を消費し buying.time からアイテムの効果を発揮する
		itemBought[b.ItemID]++
		m := mItems[b.ItemID]
		totalMilliIsu.Sub(totalMilliIsu, new(big.Int).Mul(m.GetPrice(b.Ordinal), bi1000))

		if b.Time <= currentTime {
			itemBuilt[b.ItemID]++
			power := m.GetPower(itemBought[b.ItemID])
			totalMilliIsu.Add(totalMilliIsu, new(big.Int).Mul(power, big.NewInt(currentTime-b.Time)))
			totalPower.Add(totalPower, power)
			itemPower[b.ItemID].Add(itemPower[b.ItemID], power)
		} else {
			buyingAt[b.Time] = append(buyingAt[b.Time], b)
		}
	}

	for _, m := range mItems {
		itemPower0[m.ItemID] = big2exp(itemPower[m.ItemID])
		itemBuilt0[m.ItemID] = itemBuilt[m.ItemID]
		price := m.GetPrice(itemBought[m.ItemID] + 1)
		itemPrice[m.ItemID] = price
		if 0 <= totalMilliIsu.Cmp(new(big.Int).Mul(price, bi1000)) {
			itemOnSale[m.ItemID] = 0 // 0 は 時刻 currentTime で購入可能であることを表す
		}
	}

	schedule := []Schedule{
		Schedule{
			Time:       currentTime,
			MilliIsu:   big2exp(totalMilliIsu),
			TotalPower: big2exp(totalPower),
		},
	}

	// currentTime から 1000 ミリ秒先までシミュレーションする
	for t := currentTime + 1; t <= currentTime+1000; t++ {
		totalMilliIsu.Add(totalMilliIsu, totalPower)
		updated := false

		// 時刻 t で発生する adding を計算する
		if a, ok := addingAt[t]; ok {
			updated = true
			totalMilliIsu.Add(totalMilliIsu, new(big.Int).Mul(str2big(a.Isu), bi1000))
		}

		// 時刻 t で発生する buying を計算する
		if _, ok := buyingAt[t]; ok {
			updated = true
			updatedID := map[int]bool{}
			for _, b := range buyingAt[t] {
				m := mItems[b.ItemID]
				updatedID[b.ItemID] = true
				itemBuilt[b.ItemID]++
				power := m.GetPower(b.Ordinal)
				itemPower[b.ItemID].Add(itemPower[b.ItemID], power)
				totalPower.Add(totalPower, power)
			}
			for id := range updatedID {
				itemBuilding[id] = append(itemBuilding[id], Building{
					Time:       t,
					CountBuilt: itemBuilt[id],
					Power:      big2exp(itemPower[id]),
				})
			}
		}

		if updated {
			schedule = append(schedule, Schedule{
				Time:       t,
				MilliIsu:   big2exp(totalMilliIsu),
				TotalPower: big2exp(totalPower),
			})
		}

		// 時刻 t で購入可能になったアイテムを記録する
		for itemID := range mItems {
			if _, ok := itemOnSale[itemID]; ok {
				continue
			}
			if 0 <= totalMilliIsu.Cmp(new(big.Int).Mul(itemPrice[itemID], bi1000)) {
				itemOnSale[itemID] = t
			}
		}
	}

	gsAdding := []*Adding{}
	for _, a := range addingAt {
		gsAdding = append(gsAdding, a)
	}

	gsItems := []Item{}
	for itemID, _ := range mItems {
		gsItems = append(gsItems, Item{
			ItemID:      itemID,
			CountBought: itemBought[itemID],
			CountBuilt:  itemBuilt0[itemID],
			NextPrice:   big2exp(itemPrice[itemID]),
			Power:       itemPower0[itemID],
			Building:    itemBuilding[itemID],
		})
	}

	gsOnSale := []OnSale{}
	for itemID, t := range itemOnSale {
		gsOnSale = append(gsOnSale, OnSale{
			ItemID: itemID,
			Time:   t,
		})
	}

	return &GameStatus{
		Adding:   gsAdding,
		Schedule: schedule,
		Items:    gsItems,
		OnSale:   gsOnSale,
	}, nil
}

func serveGameConn(ws *websocket.Conn, roomName string) {
	log.Println(ws.RemoteAddr(), "serveGameConn", roomName)
	defer ws.Close()

	muxByRoomNameMu.Lock()
	if _, ok := muxByRoomName[roomName]; !ok {
		muxByRoomName[roomName] = new(sync.Mutex)
	}
	muxByRoomNameMu.Unlock()

	status, err := getStatus(roomName)
	if err != nil {
		log.Println(err)
		return
	}

	err = ws.WriteJSON(status)
	if err != nil {
		log.Println(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chReq := make(chan GameRequest)

	go func() {
		defer cancel()
		for {
			req := GameRequest{}
			err := ws.ReadJSON(&req)
			if err != nil {
				log.Println(err)
				if err == io.EOF {
					leaveMemberToRoom(roomName)
				}
				return
			}

			select {
			case chReq <- req:
			case <-ctx.Done():
				return
			}
		}
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case req := <-chReq:
			log.Println(req)

			success := false
			switch req.Action {
			case "addIsu":
				success = addIsu(roomName, str2big(req.Isu), req.Time)
			case "buyItem":
				success = buyItem(roomName, req.ItemID, req.CountBought, req.Time)
			default:
				log.Println("Invalid Action")
				return
			}

			if success {
				// GameResponse を返却する前に 反映済みの GameStatus を返す
				status, err := getStatus(roomName)
				if err != nil {
					log.Println(err)
					return
				}

				err = ws.WriteJSON(status)
				if err != nil {
					log.Println(err)
					return
				}
			}

			err := ws.WriteJSON(GameResponse{
				RequestID: req.RequestID,
				IsSuccess: success,
			})
			if err != nil {
				log.Println(err)
				return
			}
		case <-ticker.C:
			status, err := getStatus(roomName)
			if err != nil {
				log.Println(err)
				return
			}

			err = ws.WriteJSON(status)
			if err != nil {
				log.Println(err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

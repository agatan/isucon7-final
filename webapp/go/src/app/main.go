package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
)

var (
	db              *sqlx.DB
	webHosts        []string
	muxByRoomNameMu sync.Mutex
	muxByRoomName   map[string]*sync.Mutex
)

func initHosts() {
	webHosts = strings.Split(os.Getenv("ISU_WEB_HOSTS"), ",")
	log.Println(webHosts)
}

func initDB() {
	db_host := os.Getenv("ISU_DB_HOST")
	if db_host == "" {
		db_host = "127.0.0.1"
	}
	db_port := os.Getenv("ISU_DB_PORT")
	if db_port == "" {
		db_port = "3306"
	}
	db_user := os.Getenv("ISU_DB_USER")
	if db_user == "" {
		db_user = "root"
	}
	db_password := os.Getenv("ISU_DB_PASSWORD")
	if db_password != "" {
		db_password = ":" + db_password
	}

	dsn := fmt.Sprintf("%s%s@tcp(%s:%s)/isudb?parseTime=true&loc=Local&charset=utf8mb4",
		db_user, db_password, db_host, db_port)

	log.Printf("Connecting to db: %q", dsn)
	db, _ = sqlx.Connect("mysql", dsn)
	for {
		err := db.Ping()
		if err == nil {
			break
		}
		log.Println(err)
		time.Sleep(time.Second * 3)
	}

	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(5 * time.Minute)
	log.Printf("Succeeded to connect db.")
}

func getInitializeHandler(w http.ResponseWriter, r *http.Request) {
	db.MustExec("TRUNCATE TABLE adding")
	db.MustExec("TRUNCATE TABLE buying")
	db.MustExec("TRUNCATE TABLE room_time")
	w.WriteHeader(204)
}

func getRoomHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	roomName := vars["room_name"]
	path := "/ws/" + url.PathEscape(roomName)

	muxByRoomNameMu.Lock()
	if _, ok := muxByRoomName[roomName]; !ok {
		muxByRoomName[roomName] = new(sync.Mutex)
	}
	host := getHostFromRoomName(roomName)
	muxByRoomNameMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Host string `json:"host"`
		Path string `json:"path"`
	}{
		Host: host,
		Path: path,
	})
}

func wsGameHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	roomName := vars["room_name"]
	addMemberToRoom(roomName)

	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		log.Println("Failed to upgrade", err)
		return
	}
	go serveGameConn(ws, roomName)
}

func main() {
	initDB()
	initRedisPool()
	initHosts()
	initRoom()
	initRoomTime()
	initMasterItems(db)
	initAddingStore()
	initBuyingStore()
	muxByRoomName = map[string]*sync.Mutex{}

	if debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		go func() {
			log.Println(http.ListenAndServe(":6060", nil))
		}()
	} else {
		log.SetOutput(ioutil.Discard)
	}

	r := mux.NewRouter()
	r.HandleFunc("/initialize", getInitializeHandler)
	r.HandleFunc("/room/", getRoomHandler)
	r.HandleFunc("/room/{room_name}", getRoomHandler)
	r.HandleFunc("/ws/", wsGameHandler)
	r.HandleFunc("/ws/{room_name}", wsGameHandler)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("../public/")))

	log.Fatal(http.ListenAndServe(":5000", handlers.LoggingHandler(os.Stderr, r)))
}

package main

import "github.com/garyburd/redigo/redis"

func addMemberToRoom(room string) {
	conn := sharedRedisPool.Get()
	defer conn.Close()

	host := getHostFromRoomName(room)
	conn.Do("ZINCRBY", "host:member_count", 1, host)
}

func leaveMemberToRoom(room string) {
	conn := sharedRedisPool.Get()
	defer conn.Close()

	host := getHostFromRoomName(room)
	conn.Do("ZINCRBY", "host:member_count", -1, host)
}

func getHostFromRoomName(room string) string {
	conn := sharedRedisPool.Get()
	defer conn.Close()

	host, err := redis.String(conn.Do("HGET", "host:room", room))
	if err != nil {
		if err == redis.ErrNil {
			hosts, err := redis.Strings(conn.Do("ZRANGE", "host:member_count", 0, 1))
			if err != nil {
				panic(err)
			}
			host := hosts[0]
			conn.Do("HSET", "host:room", room, host)
			return host
		}
		panic(err)
	}
	return host
}

func initRoom() {
	conn := sharedRedisPool.Get()
	defer conn.Close()
	var err error

	err = conn.Send("MULTI")
	if err != nil {
		panic(err)
	}
	for _, h := range webHosts {
		err = conn.Send("ZADD", "host:member_count", 0, h)
		if err != nil {
			panic(err)
		}
	}
	_, err = conn.Do("EXEC")
	if err != nil {
		panic(err)
	}
}

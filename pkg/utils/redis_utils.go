// Copyright 2015 Reborndb Org. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.

package utils

import (
	"net"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/juju/errors"
)

const RedisConnTimeoutSecond = 1

func newRedisConn(addr string, connectTimeout time.Duration, readTimeout time.Duration, writeTimeout time.Duration, password string) (redis.Conn, error) {
	c, err := redis.DialTimeout("tcp", addr, connectTimeout, readTimeout, writeTimeout)
	if err != nil {
		return nil, err
	}

	if len(password) > 0 {
		if _, err = c.Do("AUTH", password); err != nil {
			c.Close()
			return nil, err
		}
	}

	return c, nil
}

// get redis's slot size
func SlotsInfo(addr string, fromSlot int, toSlot int, password string) (map[int]int, error) {
	c, err := newRedisConn(addr, RedisConnTimeoutSecond*time.Second, RedisConnTimeoutSecond*time.Second, RedisConnTimeoutSecond*time.Second, password)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	var (
		reply []interface{}
		val   []interface{}
	)

	reply, err = redis.Values(c.Do("SLOTSINFO", fromSlot, toSlot-fromSlot+1))
	if err != nil {
		return nil, err
	}

	ret := map[int]int{}
	for {
		if reply == nil || len(reply) == 0 {
			break
		}
		if reply, err = redis.Scan(reply, &val); err != nil {
			return nil, err
		}
		var slot, keyCount int
		_, err := redis.Scan(val, &slot, &keyCount)
		if err != nil {
			return nil, err
		}
		ret[slot] = keyCount
	}

	return ret, nil
}

func GetRedisStat(addr string, password string) (map[string]string, error) {
	c, err := newRedisConn(addr, RedisConnTimeoutSecond*time.Second, RedisConnTimeoutSecond*time.Second, RedisConnTimeoutSecond*time.Second, password)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	ret, err := redis.String(c.Do("INFO"))
	if err != nil {
		return nil, err
	}

	m := map[string]string{}
	lines := strings.Split(ret, "\n")
	for _, line := range lines {
		kv := strings.SplitN(line, ":", 2)
		if len(kv) == 2 {
			k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
			m[k] = v
		}
	}

	var reply []string

	reply, err = redis.Strings(c.Do("CONFIG", "GET", "MAXMEMORY"))
	if err != nil {
		return nil, err
	}

	// we got result
	if len(reply) == 2 {
		if reply[1] != "0" {
			m["maxmemory"] = reply[1]
		} else {
			m["maxmemory"] = "∞"
		}
	}

	return m, nil
}

func GetRedisConfig(addr string, configName string, password string) (string, error) {
	c, err := newRedisConn(addr, RedisConnTimeoutSecond*time.Second, RedisConnTimeoutSecond*time.Second, RedisConnTimeoutSecond*time.Second, password)
	if err != nil {
		return "", err
	}
	defer c.Close()

	ret, err := redis.Strings(c.Do("CONFIG", "GET", configName))
	if err != nil {
		return "", err
	}

	if len(ret) == 2 {
		return ret[1], nil
	}

	return "", nil
}

func SlaveOf(slave string, master string, password string, masterPassword string) error {
	if master == slave {
		return errors.New("can not slave of itself")
	}

	c, err := newRedisConn(slave, RedisConnTimeoutSecond*time.Second, RedisConnTimeoutSecond*time.Second, RedisConnTimeoutSecond*time.Second, password)
	if err != nil {
		return errors.Trace(err)
	}
	defer c.Close()

	host, port, err := net.SplitHostPort(master)
	if err != nil {
		return errors.Trace(err)
	}

	// Todo
	// Maybe we should set master password for slave

	_, err = c.Do("SLAVEOF", host, port)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func SlaveNoOne(addr string, password string) error {
	c, err := newRedisConn(addr, RedisConnTimeoutSecond*time.Second, RedisConnTimeoutSecond*time.Second, RedisConnTimeoutSecond*time.Second, password)
	if err != nil {
		return errors.Trace(err)
	}
	defer c.Close()

	_, err = c.Do("SLAVEOF", "NO", "ONE")
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

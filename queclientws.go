package quev2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	bolt "go.etcd.io/bbolt"
)

const (
	WS_ACK           = "ack"
	WS_ASSIGNED      = "assigned"
	WS_SUBOK         = "sub_ok"
	WS_SUB           = "sub"
	WS_UNSUB         = "un_sub"
	WS_WELCOME       = "welcome"
	WS_ROUTE         = "route"
	WS_PAUSE         = "pause"
	WS_UNPAUSE       = "unpause"
	WS_PAUSE_STATUS  = "pause_status"
	WS_MUTEX_CONFIRM = "mutex_confirm"
)

var (
	missedItems []*QueItem
)

var cacheFilePath = "/Users/faisaliqbal/queV2Cache"

func init() {
	if runtime.GOOS == "linux" {
		cacheFilePath = "/var/faisaliqbal/queV2Cache"
	}
	err := os.MkdirAll(cacheFilePath, 0777)
	if err != nil {
		panic(err)
	}
}

type wsCache struct {
	db *bolt.DB
}

func (c *wsCache) open(pc string) {
	fn := filepath.Join(cacheFilePath, "channel_"+pc+".dat")
	var err error
	c.db, err = bolt.Open(fn, 0777, nil)
	if err != nil {
		panic(err)
	}
}

func (c *wsCache) close() {
	if c.db != nil {
		c.db.Close()
	}
}

func (c *wsCache) Get(id string) *WorkitemStatus {
	var one WorkitemStatus
	c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("_default_"))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(id))
		if len(v) == 0 {
			return nil
		}
		json.Unmarshal(v, &one)
		return nil
	})
	if len(one.ID) == 0 {
		return nil
	}
	return &one
}

func (c *wsCache) Set(w *WorkitemStatus) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("_default_"))
		if err != nil {
			return err
		}
		w.LastUpdated = time.Now()
		jsoned, _ := json.Marshal(w)
		return b.Put([]byte(w.ID), jsoned)
	})
}

func (c *wsCache) Del(id string) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("_default_"))
		if b == nil {
			return nil
		}

		return b.Delete([]byte(id))
	})
}

func (c *wsCache) CleanUp(since time.Time) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("_default_"))
		if b == nil {
			return nil
		}
		cursor := b.Cursor()
		for {
			k, v := cursor.Next()
			if len(k) == 0 || len(v) == 0 {
				break
			}
			var one WorkitemStatus
			json.Unmarshal(v, &one)
			if one.LastUpdated.Before(since) {
				cursor.Delete()
			}
		}
		return nil
	})
}

const (
	WorkItemStatusInitial = iota
	WorkItemStatusSuccess
	WorkItemStatusError
)

type WorkitemStatus struct {
	ID          string
	ASKSend     bool
	Status      int
	FirstAdded  time.Time
	LastUpdated time.Time
}

type Wsclient struct {
	url            string
	conn           *websocket.Conn
	C              chan *QueItem
	urlC           url.URL
	connected      bool
	isExiting      bool
	PrimaryChannel string
	cache          *wsCache
	connecting     bool
	ExitC          chan os.Signal
	Subscribed     bool
	IsPaused       bool
}

func (ws *Wsclient) Close() {
	ws.isExiting = true
	if ws.conn != nil {
		ws.conn.Close()
	}
}

func (ws *Wsclient) SendACK(qi *QueItem) {
	ws.Send(&QueItem{ID: qi.ID, Action: WS_ACK})
}

func (ws *Wsclient) Send(qi *QueItem) error {

	if ws.connecting || ws.conn == nil {
		missedItems = append(missedItems, qi)
		return errors.New("connection is no in connected state")
	}

	if qi.Action == WS_ROUTE {
		qi.MutexConfirmed = false
	}

	jsoned, _ := json.Marshal(qi)

	err := ws.conn.WriteMessage(websocket.TextMessage, jsoned)
	if err != nil {
		missedItems = append(missedItems, qi)
		return err
	}

	if qi.Action == WS_ACK {
		wi := ws.cache.Get(qi.ID)
		wi.ASKSend = true
		ws.cache.Set(wi)
	}
	return nil
}

func (ws *Wsclient) retryMissedItems() error {
	for len(missedItems) > 0 {
		missed := missedItems[0]
		missedItems = missedItems[1:]
		err := ws.Send(missed)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ws *Wsclient) _connect() error {
	ws.connecting = true

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	c, _, err := websocket.DefaultDialer.DialContext(ctx, ws.urlC.String(), nil)
	if err != nil {
		return err
	}

	ws.connected = true
	ws.connecting = false
	ws.conn = c
	ws.retryMissedItems()
	return nil
}

func (ws *Wsclient) CleanUpCache(since time.Time) {
	ws.cache.CleanUp(since)
}

func (ws *Wsclient) Route(i *QueItem, target string) error {
	i.Channel = target
	i.Action = WS_ROUTE
	return ws.Send(i)
}

func (ws *Wsclient) CacheSuccess(id string) error {
	one := ws.cache.Get(id)
	if one == nil {
		return nil
	}
	one.Status = WorkItemStatusSuccess
	return ws.cache.Set(one)
}

func (ws *Wsclient) Exiting() {
	ws.isExiting = true
}

func (ws *Wsclient) CacheFailure(id string) error {
	one := ws.cache.Get(id)
	if one == nil {
		return nil
	}
	one.Status = WorkItemStatusError
	return ws.cache.Set(one)
}

func (ws *Wsclient) Subscribe(channel string) error {
	return ws.Send(&QueItem{Action: WS_SUB, Channel: channel})
}

func (ws *Wsclient) Open(urls ...string) error {
	ws.C = make(chan *QueItem, 1)
	ws.ExitC = make(chan os.Signal, 1)

	ws.cache = &wsCache{}
	ws.cache.open(ws.PrimaryChannel)

	signal.Notify(ws.ExitC, os.Kill, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)

	var wsURL = "localhost:7021"
	if len(ws.url) > 0 {
		wsURL = ws.url
	}

	if len(urls) > 0 {
		wsURL = urls[0]
		ws.url = urls[0]
	}

	u := url.URL{Scheme: "ws", Host: wsURL, Path: "/ws"}
	ws.urlC = u
	ws._connect()
	go ws.reader()
	return nil
}

func (ws *Wsclient) reader() {
	defer ws.cache.close()

	for {
		select {
		case <-ws.ExitC:
			fmt.Println("Exiting from WSClient")
			ws.isExiting = true
			ws.connected = false
			return

		default:
			_, msg, err := ws.conn.ReadMessage()
			if err != nil {
				fmt.Println("Connection Ended", err)
				if ws.isExiting {
					ws.connected = false
					return
				}

				time.Sleep(time.Second * 1)
				fmt.Println("Going to try to reconnect")
				ws._connect()
				continue
			}

			var one QueItem
			json.Unmarshal(msg, &one)

			if one.Action == WS_WELCOME {
				ws.Send(&QueItem{Action: WS_SUB, Channel: ws.PrimaryChannel})
				continue
			}

			if one.Action == WS_PAUSE {
				ws.IsPaused = true
			}

			if one.Action == WS_UNPAUSE {
				ws.IsPaused = false
			}

			if one.Action == WS_SUBOK {
				if one.Channel == ws.PrimaryChannel {
					ws.Subscribed = true
				}
			}

			if one.Action == WS_ASSIGNED {
				if !one.MutexConfirmed {
					one.Action = WS_MUTEX_CONFIRM
					ws.Send(&one)
					continue
				}

				wi := ws.cache.Get(one.ID)
				if wi == nil {
					wi = &WorkitemStatus{ID: one.ID, ASKSend: false, Status: WorkItemStatusInitial, FirstAdded: time.Now(), LastUpdated: time.Now()}
					ws.cache.Set(wi)
				}

				if wi.Status == WorkItemStatusError || wi.Status == WorkItemStatusSuccess {
					ws.Send(&QueItem{Action: WS_ACK, ID: wi.ID})
					continue
				}
			}

			ws.C <- &one
		}
	}
}

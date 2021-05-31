package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sfi2k7/quev2"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/redis.v4"
)

var lastMessageInOutgoing *quev2.QueItem

var (
	connections    *mmap
	routingChannel chan *quev2.QueItem
	red            *redis.Client
	dir            *subsdirs
)

func init() {
	connections = newmmap()
	// incomingChannel = make(chan *IncomingMessage, 200)
	routingChannel = make(chan *quev2.QueItem, 200)

	red = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		DB:       0,
		Network:  "tcp",
		Password: "passme",
	})

	if p, err := red.Ping().Result(); p != "PONG" {
		panic(err)
	}
	dir = newSubsDirs()
}

func newRedis() (*redis.Client, error) {
	c := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		DB:       0,
		Network:  "tcp",
		Password: "passme",
	})

	if c == nil {
		return nil, errors.New("Redis Client is Nil")
	}

	if _, err := c.Ping().Result(); err != nil {
		return nil, err
	}
	return c, nil
}

var upgrader = websocket.Upgrader{EnableCompression: false, HandshakeTimeout: time.Second * 5, ReadBufferSize: 4096, WriteBufferSize: 4096}

func Upgrade(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		fmt.Println(err)
		return
	}

	now := time.Now()
	dh := &DeviceHandler{c: c, w: make(chan *quev2.QueItem, 1000), ConnectionAt: &now}
	defer dh.close()
	dh.ID = UId()

	connections.add(dh.ID, dh)
	fmt.Println(" USER CONNECTED ", connections.count())

	dh.handle()

	connections.remove(dh.ID)
}

func feeder(ctx context.Context) {
	//Pick messages from DB and try to feed to subscriber
	//Using Outgoing channel
	//Pick 10 messages every 1 minutes
	//Do not pick the same message in 10 minutes
	//Duplicate is handled by Mutex in each component
	processed := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
			list, err := quev2.GetMultiQueItem(time.Now().Add(-(time.Second * 10)))
			if err != nil {
				time.Sleep(time.Second * 1)
				continue
			}

			if len(list) == 0 {
				time.Sleep(time.Second * 1)
				continue
			}

			for _, one := range list {

				if one.Action != "route" {
					quev2.RemoveQueItem(one.ID)
					continue
				}

				routingChannel <- one
				quev2.UpdateLastFeed(one.ID)
			}

			processed = processed + len(list)

			if processed > 100 {
				time.Sleep(time.Second * 1)
				processed = 0
			}
		}
	}
}

func outgoingProcess(w chan *quev2.QueItem) {
	for {
		m, ok := <-w
		if !ok {
			return
		}

		if m == nil {
			return
		}

		if m.Action == quev2.WS_ACK {
			err := quev2.RemoveQueItem(m.ID)
			if err != nil {
				fmt.Println("Err returned by  Remove Que Item", err)
			}

			continue
		}

		if m.Action == quev2.WS_ROUTE {
			lastMessageInOutgoing = m
			id, err := quev2.AddQueItem(m)
			if err != nil {
				fmt.Println("Adding Que Item", err)
			}

			m.ID = id

			if dir.isChannelPaused(m.Channel) {
				continue
			}

			//Check Channel pause status
			subs := dir.getsubs(m.Channel)
			for _, sub := range subs {
				if sub.exiting || sub.isClosed || len(sub.ID) == 0 {
					dir.remove(sub.ID)
					continue
				}

				m.Action = quev2.WS_ASSIGNED

				if len(sub.w) < 100 {
					sub.w <- m
				}

			}
			continue
		}
		fmt.Println("Unknown Message in Outgoing", m.Action)
	}
}

//MUTEX Protocol
/*
	Assign
	ASK
	MUTEX_CHECK
		CHECK MUTEX_LIST
			IF NONE_EXISTS:
				ADD ITEM BACK INTO Q
			ELSE:
				DELETE ITEM

*/
func Router(items []*quev2.QueItem) {
	byChannel := make(map[string][]*quev2.QueItem)
	for _, item := range items {
		subs, ok := byChannel[item.Channel]
		if !ok {
			subs = make([]*quev2.QueItem, 0)
			byChannel[item.Channel] = subs
		}
		subs = append(subs, item)
		byChannel[item.Channel] = subs
	}

	//LOGIC:
	/*

		Divide messages into channels
		for channel:
			subs := get subscribers
			each messages per channel
				x = 0
				sub = subs[x]
				send message to sub
				x++
				if len(x) > len(subs)
				x = 0

	*/
}

func cleanupConnections(c context.Context) {
	tmr := time.NewTimer(time.Minute * 1)
	defer tmr.Stop()

	for {
		select {
		case <-tmr.C:
			dir.cleanupLingeringConnections()
		case <-c.Done():
			return
		}
	}
}

func getStats(w http.ResponseWriter, r *http.Request) {
	subscribers := dir.getSubCounts()

	channelStatus := make(map[string]bool)

	for ch := range subscribers {
		channelStatus[ch] = dir.isChannelPaused(ch)
	}

	_ = bson.M{
		"subCount":           subscribers,
		"lastItemInOutgoing": lastMessageInOutgoing,
		"connectionCount":    connections.count(),
		"channelPauseStatus": channelStatus,
		"itemsInQue":         quev2.GetQueCount(),
	}

	http.ServeFile(w, r, "./public/index.html")
	// t, err := template.New("index.html").ParseFiles("./public/index.html")
	// if err != nil {
	// 	w.Write([]byte(err.Error()))
	// 	return
	// }
	// t.Execute(w, data)
}

func sendToChannel(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	ch := q.Get("ch")
	routingChannel <- &quev2.QueItem{Action: quev2.WS_ROUTE, Channel: ch}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func channelPauser(w http.ResponseWriter, r *http.Request) {
	ch := r.URL.Query().Get("ch")
	pause := r.URL.Query().Get("p")
	subs := dir.getSubCounts()
	for k := range subs {
		if len(ch) == 0 || ch == "all" || ch == k || strings.Contains(ch, k) {
			if pause == "1" || pause == "true" || pause == "yes" || pause == "pause" {
				fmt.Println("pausing channel", k)
				dir.pauseChannel(k)
			} else {
				fmt.Println("un-pausing channel", k)
				dir.unPauseChannel(k)
			}
		}
	}
	w.Write([]byte("OK"))
}

func main() {
	one := make(chan os.Signal, 1)

	signal.Notify(one, os.Kill, os.Interrupt)

	upgrader.CheckOrigin = func(r *http.Request) bool {
		//TODO: Check Source
		//fmt.Println("CALLFROM", r.Header.Get("Origin"))
		return true
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctxfeeder, cancel := context.WithCancel(context.Background())
	defer cancel()

	go cleanupConnections(ctx)
	go outgoingProcess(routingChannel)
	go feeder(ctxfeeder)

	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/public/", http.StripPrefix("/public/", fs))
	http.HandleFunc("/ws", Upgrade)
	http.HandleFunc("/stats", getStats)
	http.HandleFunc("/route", sendToChannel)
	http.HandleFunc("/pauser", channelPauser)

	var server http.Server
	server = http.Server{Addr: ":7021", Handler: nil}

	go server.ListenAndServe()

	<-one

	close(routingChannel)

	server.Shutdown(context.Background())
}

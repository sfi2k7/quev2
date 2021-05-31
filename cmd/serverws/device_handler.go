package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/sfi2k7/quev2"

	"github.com/gorilla/websocket"
)

type DeviceHandler struct {
	c            *websocket.Conn
	w            chan *quev2.QueItem
	ex           chan bool
	exiting      bool
	isClosed     bool
	ID           string
	ConnectionAt *time.Time
}

func (dh *DeviceHandler) handle() {
	defer func() {
		dh.isClosed = true
	}()

	dh.exiting = false
	dh.ex = make(chan bool, 2)

	go func() {
		for s := range dh.w {
			jsoned, _ := json.Marshal(s)
			err := dh.c.WriteMessage(websocket.TextMessage, jsoned)
			if dh.exiting {
				return
			}

			if err != nil {
				dh.exiting = true
				dh.ex <- true
				return
			}
		}
	}()

	go func() {
		for {
			if dh.c == nil {
				return
			}

			_, b, err := dh.c.ReadMessage()
			if dh.exiting {
				return
			}

			if err != nil || len(b) == 0 {
				dh.exiting = true
				dh.ex <- true
				return
			}

			var qitem quev2.QueItem
			json.Unmarshal(b, &qitem)

			fmt.Println("->", qitem.Action, qitem.Channel, qitem.MutexConfirmed)
			if qitem.Action == quev2.WS_SUB {
				dir.add(qitem.Channel, dh)
				dh.w <- &quev2.QueItem{
					Action:  quev2.WS_SUBOK,
					Channel: qitem.Channel,
				}
				continue
			}

			if qitem.Action == quev2.WS_MUTEX_CONFIRM {
				err := quev2.MutexConfirm(&qitem)
				if err != nil {
					continue
				}

				qitem.Action = quev2.WS_ASSIGNED
				qitem.MutexConfirmed = true
				dh.w <- &qitem
				continue
			}

			if qitem.Action == quev2.WS_UNSUB {
				dir.remove(dh.ID)
				continue
			}

			routingChannel <- &qitem
			// incomingChannel <- &msg
			// fmt.Println("Unknown Message", qitem)
		}
	}()
	dh.w <- &quev2.QueItem{Action: "welcome", Channel: "ws", Data: "QueV2 Server"}
	<-dh.ex
	fmt.Println("DX Exited")
	dir.remove(dh.ID)
}

func (dh *DeviceHandler) close() {
	if dh.c != nil {
		dh.c.Close()
		dh.c = nil
	}

	if dh.w != nil {
		close(dh.w)
		dh.w = nil
	}

	if dh.ex != nil {
		close(dh.ex)
		dh.ex = nil
	}

	//incomingQueue.Send(&cpq.Message{Command: "CLOSE", DeviceId: dh.deviceId, Source: "HUB"})
	//cpq.Send(QIncomingName, &cpq.Message{Command: "CLOSE", DeviceId: dh.deviceId, Source: "HUB"})
}

func UId() string {
	u := uuid.NewV4()
	return strings.Replace(u.String(), "-", "", -1)
}

func newDeviceHandler() *DeviceHandler {
	u := uuid.NewV4()
	id := strings.Replace(u.String(), "-", "", -1)
	return &DeviceHandler{
		ID: id,
	}
}

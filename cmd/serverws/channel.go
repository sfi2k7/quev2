package main

import (
	"fmt"
	"sync"

	"github.com/sfi2k7/quev2"
)

var (
	_channelLock sync.Mutex
)

func init() {
	_channelLock = sync.Mutex{}
}

type subsdirs struct {
	subs   map[string]map[string]*DeviceHandler
	cmap   map[string][]string
	pauses map[string]string
}

func (sd *subsdirs) pauseChannel(ch string) {
	_channelLock.Lock()
	defer _channelLock.Unlock()

	sd.pauses[ch] = quev2.WS_PAUSE
}

func (sd *subsdirs) unPauseChannel(ch string) {
	_channelLock.Lock()
	defer _channelLock.Unlock()

	sd.pauses[ch] = quev2.WS_UNPAUSE
}

func (sd *subsdirs) isChannelPaused(ch string) bool {
	status, ok := sd.pauses[ch]
	if !ok {
		return false
	}
	return status == quev2.WS_PAUSE
}

func (sd *subsdirs) add(channel string, dh *DeviceHandler) {
	_channelLock.Lock()
	defer _channelLock.Unlock()

	_, ok := sd.subs[channel]
	if !ok {
		sd.subs[channel] = make(map[string]*DeviceHandler)
	}

	sd.subs[channel][dh.ID] = dh
	if _, ok := sd.cmap[dh.ID]; !ok {
		sd.cmap[dh.ID] = make([]string, 0)
	}

	uc := make(map[string]struct{})
	for _, c := range sd.cmap[dh.ID] {
		uc[c] = struct{}{}
	}

	uc[channel] = struct{}{}
	sd.cmap[dh.ID] = make([]string, 0)
	for channel := range uc {
		sd.cmap[dh.ID] = append(sd.cmap[dh.ID], channel)
	}

	fmt.Println("Adding", dh.ID, "To", channel, len(sd.subs[channel]), len(sd.cmap[dh.ID]))
}

func (sd *subsdirs) remove(id string) {
	_channelLock.Lock()
	defer _channelLock.Unlock()
	fmt.Println("Removeing", id)
	channels, ok := sd.cmap[id]

	if !ok {
		return
	}

	for _, ch := range channels {
		delete(sd.subs[ch], id)
	}

	delete(sd.cmap, id)
}

func (sd *subsdirs) getsubs(channel string) []*DeviceHandler {
	_channelLock.Lock()
	defer _channelLock.Unlock()

	subs, ok := sd.subs[channel]
	if !ok {
		return nil
	}

	var list []*DeviceHandler
	for _, sub := range subs {
		if sub == nil || sub.isClosed || sub.exiting {
			continue
		}

		list = append(list, sub)
	}
	return list
}

func (sd *subsdirs) getSubCounts() map[string]int {
	_channelLock.Lock()
	defer _channelLock.Unlock()

	sc := make(map[string]int)

	for channel, each := range sd.subs {
		if _, ok := sc[channel]; !ok {
			sc[channel] = 0
		}
		sc[channel] = len(each)
	}
	return sc
}

func (sd *subsdirs) cleanupLingeringConnections() {
	_channelLock.Lock()
	defer _channelLock.Unlock()

	for _, subs := range sd.subs {
		for subId, sub := range subs {
			if sub == nil || sub.exiting || sub.isClosed || sub.c == nil {
				delete(subs, subId)
				delete(sd.cmap, subId)
			}
		}
	}
}

func newSubsDirs() *subsdirs {
	return &subsdirs{
		subs:   make(map[string]map[string]*DeviceHandler),
		cmap:   make(map[string][]string),
		pauses: make(map[string]string),
	}
}

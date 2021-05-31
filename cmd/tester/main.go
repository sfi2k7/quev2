package main

import (
	"flag"
	"fmt"

	"github.com/sfi2k7/quev2"
)

func main() {
	channelS := flag.String("ch", "", "Channel to subscribe")
	nextChS := flag.String("next", "", "next channel in chain")
	flag.Parse()

	if len(*channelS) == 0 {
		panic("Must specify a valid channel name using --ch flag")
	}

	// if len(*nextChS) == 0 {
	// 	panic("Must specify a valid Next channel name using --next flag")
	// }

	var channel = *channelS

	ws := quev2.Wsclient{PrimaryChannel: channel}

	ws.Open()

	for {
		select {
		case <-ws.ExitC:
			ws.Exiting()
			fmt.Println("Exiting from tester")
			return
		case qi := <-ws.C:
			fmt.Println("Item Arrived", qi)

			if qi.Action == quev2.WS_ASSIGNED {
				ws.SendACK(qi)
				fmt.Println("Processing", qi.ID)
				ws.CacheSuccess(qi.ID)

				if len(*nextChS) > 0 {
					ws.Route(qi, *nextChS)
					fmt.Println("rerouting to ", *nextChS)
				}
			}
		}
	}
}

func main1() {
	q := quev2.QueClient{}
	// err := q.Move("need_sent", "done_sent", "123")
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(items)

	items, err := q.Get("need_sent")
	if err != nil {
		panic(err)
	}
	fmt.Println(items)

	items, err = q.Get("done_sent")
	if err != nil {
		panic(err)
	}
	fmt.Println(items)
}

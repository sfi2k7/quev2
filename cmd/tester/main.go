package main

import (
	"fmt"

	"github.com/sfi2k7/quev2"
)

func main() {
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

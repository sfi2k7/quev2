package main

import (
	"time"

	"github.com/sfi2k7/picoweb"
	"github.com/sfi2k7/quev2"
)

var (
	qdb *quev2.Quedb
)

func getitem(c *picoweb.Context) {
	l := c.Query("l")
	if len(l) == 0 {
		c.Json(&quev2.ApiResponse{Error: "Must specify a listnane", Took: time.Since(c.Start).String()})
		return
	}
	items, err := qdb.Get(l)
	if err != nil {
		c.Json(&quev2.ApiResponse{Error: err.Error(), Took: time.Since(c.Start).String()})
		return
	}
	c.Json(&quev2.ApiResponse{Result: items, Success: true, Took: time.Since(c.Start).String()})
}

func setitem(c *picoweb.Context) {
	l := c.Query("l")
	item := c.Query("i")

	if len(l) == 0 || len(item) == 0 {
		c.Json(&quev2.ApiResponse{Error: "Must specify a listnane AND Item", Took: time.Since(c.Start).String()})
		return
	}

	err := qdb.Set(l, item)

	if err != nil {
		c.Json(&quev2.ApiResponse{Error: err.Error(), Took: time.Since(c.Start).String()})
		return
	}
	c.Json(&quev2.ApiResponse{Success: true, Took: time.Since(c.Start).String()})
}

func deleteitem(c *picoweb.Context) {
	l := c.Query("l")
	item := c.Query("i")

	if len(l) == 0 || len(item) == 0 {
		c.Json(&quev2.ApiResponse{Error: "Must specify a listnane AND Item", Took: time.Since(c.Start).String()})
		return
	}

	err := qdb.Remove(l, item)

	if err != nil {
		c.Json(&quev2.ApiResponse{Error: err.Error(), Took: time.Since(c.Start).String()})
		return
	}
	c.Json(&quev2.ApiResponse{Success: true, Took: time.Since(c.Start).String()})
}

func moveitem(c *picoweb.Context) {
	tl := c.Query("tl")
	item := c.Query("i")

	if len(tl) == 0 || len(item) == 0 {
		c.Json(&quev2.ApiResponse{Error: "Must specify both from and to listnanes AND Item", Took: time.Since(c.Start).String()})
		return
	}

	err := qdb.Move(tl, item)

	if err != nil {
		c.Json(&quev2.ApiResponse{Error: err.Error(), Took: time.Since(c.Start).String()})
		return
	}
	c.Json(&quev2.ApiResponse{Success: true, Took: time.Since(c.Start).String()})
}

func main() {
	var err error
	qdb, err = quev2.NewQDB()
	if err != nil {
		panic(err)
	}

	web := picoweb.New()
	web.Get("/", getitem)
	web.Post("/", setitem)
	web.Delete("/", deleteitem)
	web.Put("/", moveitem)
	web.StopOnInt()
	web.Production()
	web.Listen(7676)
}

package quev2

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	qv2DB = "blue_que_v2"
)

const (
	queitemscname = "items"
	mutexcname    = "mutexes"
)

func Mongo() *mgo.Session {
retry:
	s, err := mgo.Dial("mongodb://localhost")
	if err != nil {
		fmt.Println("Could not connect to Database, Remember Everyone trying in 1 second", err)
		time.Sleep(time.Second * 1)
		goto retry
	}

	return s
}

type QueItem struct {
	ID             string      `json:"id" bson:"_id"`
	Action         string      `json:"action" bson:"action"`
	Channel        string      `json:"channel" bson:"channel"`
	Data           interface{} `json:"data" bson:"data"`
	Added          time.Time   `json:"added" bson:"added"`
	LastUpdated    time.Time   `json:"lastUpdated" bson:"lastUpdated"`
	Chain          string      `json:"chain" bson:"chain"`
	LastFeed       time.Time   `json:"lastFeed" bson:"lastFeed"`
	MutexConfirmed bool        `json:"mutexConfirmed" bson:"mutexConfirmed"`
	Priority       int         `json:"priority" bson:"priority"`
}

func MutexConfirm(qi *QueItem) error {
	fmt.Println("Mutex Checking", qi.ID)
	s := Mongo()
	defer s.Close()

	err := s.DB(qv2DB).C(mutexcname).Insert(bson.M{"_id": qi.ID + "_" + qi.Channel})
	if err != nil {
		str := err.Error()
		if strings.Contains(str, "duplicate") {
			RemoveQueItem(qi.ID)
			return err
		}
		return err
	}

	return nil
}

func AddQueItem(qi *QueItem) (string, error) {
	s := Mongo()
	defer s.Close()

	if len(qi.ID) == 0 {
		qi.ID = NewV4()
	}

	err := s.DB(qv2DB).C(queitemscname).Insert(qi)
	return qi.ID, err
}

func GetQueCount() int {
	s := Mongo()
	defer s.Close()

	n, _ := s.DB(qv2DB).C(queitemscname).Count()
	return int(n)
}

func RemoveQueItem(id string) error {
	s := Mongo()
	defer s.Close()

	err := s.DB(qv2DB).C(queitemscname).RemoveId(id)
	if err != nil && err == mgo.ErrNotFound {
		return nil
	}

	return err
}

func GetOneQueItem() (*QueItem, error) {
	s := Mongo()
	defer s.Close()
	var one QueItem
	err := s.DB(qv2DB).C(queitemscname).Find(bson.M{}).Limit(1).One(&one)
	return &one, err
}

func GetMultiQueItem(since time.Time) ([]*QueItem, error) {
	var list []*QueItem

	s := Mongo()
	defer s.Close()

	err := s.DB(qv2DB).C(queitemscname).Find(bson.M{
		"$or": []bson.M{
			{"lastFeed": bson.M{"$lte": since}},
			{"lastFeed": bson.M{"$exists": false}},
		}},
	).All(&list)
	return list, err
}

func UpdateLastFeed(id string) error {
	s := Mongo()
	defer s.Close()

	err := s.DB(qv2DB).C(queitemscname).UpdateId(id, bson.M{"$set": bson.M{"lastFeed": time.Now()}})
	if err != nil && err == mgo.ErrNotFound {
		return nil
	}

	return err
}

// func GetMessageFromChannelUnreserved(channel string) (*QueItem, error) {
// 	s := Mongo()
// 	defer s.Close()

// 	var one QueItem
// 	err := s.DB(qv2DB).C(queitemscname).Find(bson.M{"channel": channel, "reserved": false}).One(&one)
// 	if err == nil && len(one.ID) > 0 {
// 		s.DB(qv2DB).C(queitemscname).UpdateId(one.ID, bson.M{"$set": bson.M{"reserved": true, "reservedAt": time.Now()}})
// 	}
// 	return &one, err
// }

func MoveMessageToChannel(id, channel string) error {
	s := Mongo()
	defer s.Close()

	err := s.DB(qv2DB).C(queitemscname).UpdateId(id, bson.M{"$set": bson.M{"channel": channel, "reserved": false}})
	return err
}

func RemoveMessage(id string) error {
	s := Mongo()
	defer s.Close()

	err := s.DB(qv2DB).C(queitemscname).RemoveId(id)
	return err
}

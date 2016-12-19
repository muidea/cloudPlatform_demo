package main

import (
	"demo/dbHelper"
	"demo/model"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/go-martini/martini"

	"strconv"

	"github.com/streadway/amqp"
)

var mongodbAddr = ""
var mongodbName = ""

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

// splitParam 分割URL参数
func splitParam(params string) map[string]string {
	result := make(map[string]string)

	for _, param := range strings.Split(params, "&") {
		items := strings.Split(param, "=")
		if len(items) == 2 {
			result[strings.ToLower(items[0])] = items[1]
		}
	}

	return result
}

func saveData(msgs <-chan amqp.Delivery) {
	dbHelper := dbHelper.NewDBHelper()
	dbHelper.Open(mongodbAddr, mongodbName)
	defer dbHelper.Close()

	// 每小时存一个collection
	collectionName := fmt.Sprintf("hisPoint_%s", time.Now().Format("2006010215"))
	currentCollection, ret := dbHelper.FetchCollection(collectionName)
	if !ret {
		panic("FetchCollection Failed")
	}
	log.Printf("create new collection,Name:%s", collectionName)

	for d := range msgs {
		curName := fmt.Sprintf("hisPoint_%s", time.Now().Format("2006010215"))
		if collectionName != curName {
			currentCollection, ret = dbHelper.FetchCollection(curName)
			if !ret {
				panic("FetchCollection Failed")
			}

			collectionName = curName
			log.Printf("create new collection,Name:%s", collectionName)
		}

		rtdData := []model.RTDData{}
		err := json.Unmarshal(d.Body, &rtdData)
		if err != nil {
			log.Print("err:" + err.Error())
			continue
		}

		// log.Printf("point size:%d", len(rtdData))
		totalSize := len(rtdData)
		offSet := 0
		const step = 5
		for true {
			if totalSize-offSet >= 5 {
				currentCollection.Insert(rtdData[offSet+0], rtdData[offSet+1], rtdData[offSet+2], rtdData[offSet+3], rtdData[offSet+4])
				offSet += step
				continue
			} else if totalSize-offSet >= 4 {
				currentCollection.Insert(rtdData[offSet+0], rtdData[offSet+1], rtdData[offSet+2], rtdData[offSet+3])
				offSet += 4
				break
			} else if totalSize-offSet >= 3 {
				currentCollection.Insert(rtdData[offSet+0], rtdData[offSet+1], rtdData[offSet+2])
				offSet += 3
				break
			} else if totalSize-offSet >= 2 {
				currentCollection.Insert(rtdData[offSet+0], rtdData[offSet+1])
				offSet += 2
				break
			} else if totalSize-offSet >= 1 {
				currentCollection.Insert(rtdData[offSet+0])
				offSet++
				break
			} else {
				// Noting todo
			}

			if offSet >= totalSize {
				break
			}
		}
	}
}

func collectionHandler(res http.ResponseWriter, req *http.Request) {
	dbHelper := dbHelper.NewDBHelper()
	dbHelper.Open(mongodbAddr, mongodbName)
	defer dbHelper.Close()

	result, _ := dbHelper.Collections()
	b, err := json.Marshal(result)
	if err != nil {
		panic("json.Marshal, failed, err:" + err.Error())
	}

	res.Write(b)
}

// CollectionHisData Collection历史数据
type CollectionHisData struct {
	ErrCode   int
	Reason    string
	Name      string
	BeginTime int64
	EndTime   int64
	Data      []model.DataValue
}

func collectionDataHandler(res http.ResponseWriter, req *http.Request) {
	dbHelper := dbHelper.NewDBHelper()
	dbHelper.Open(mongodbAddr, mongodbName)
	defer dbHelper.Close()

	result := CollectionHisData{BeginTime: 0, EndTime: 0}
	params := splitParam(req.URL.RawQuery)
	for true {
		collectionName, found := params["collection"]
		if !found {
			result.ErrCode = 1
			result.Reason = "请指定Collection"
			break
		}
		pointName, found := params["point"]
		if !found {
			result.ErrCode = 1
			result.Reason = "请指定Point"
			break
		}
		bTime, found := params["begintime"]
		if !found {
			result.ErrCode = 1
			result.Reason = "请指定beginTime"
			break
		}
		beginTime, err := strconv.Atoi(bTime)
		if err != nil {
			result.ErrCode = 1
			result.Reason = "非法beginTime"
			break
		}
		eTime, found := params["endtime"]
		if !found {
			result.ErrCode = 1
			result.Reason = "请指定endTime"
			break
		}
		endTime, err := strconv.Atoi(eTime)
		if err != nil {
			result.ErrCode = 1
			result.Reason = "非法endTime"
			break
		}

		collection, found := dbHelper.FetchCollection(collectionName)
		if !found {
			result.ErrCode = 1
			result.Reason = "无效Collection参数"
			break
		}

		rtdData := []model.RTDData{}
		collection.Find(bson.M{"name": pointName}).All(&rtdData)
		log.Printf("filter data, Collection:%s, Name:%s, beginTime:%d, endTime:%d, count:%d", collectionName, pointName, beginTime, endTime, len(rtdData))
		for _, val := range rtdData {
			if val.TimeStamp >= int64(beginTime) && val.TimeStamp <= int64(endTime) {
				if result.BeginTime == 0 {
					result.BeginTime = val.TimeStamp
				}

				result.EndTime = val.TimeStamp
				pointVal := model.DataValue{Value: val.Value, Quality: val.Quality, TimeStamp: val.TimeStamp}
				result.Data = append(result.Data, pointVal)
			}
		}
		result.Name = pointName

		break
	}

	b, err := json.Marshal(result)
	if err != nil {
		panic("json.Marshal, failed, err:" + err.Error())
	}

	res.Write(b)
}

func main() {
	var rabbitmq = ""
	flag.StringVar(&rabbitmq, "Rabbitmq", "amqp://guest:guest@localhost:5672/", "rabbitmq address")
	flag.StringVar(&mongodbAddr, "DataBaseSvr", "127.0.0.1", "mongodb server address")
	flag.StringVar(&mongodbName, "DataBaseName", "cloudPlatform-mongodb", "mongodb database name")
	flag.Parse()

	conn, err := amqp.Dial(rabbitmq)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"rtdPacket", // name
		false,       // durable
		false,       // delete when usused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	go saveData(msgs)

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	m := martini.Classic()

	m.Get("/collection/", collectionHandler)

	m.Get("/collection/data/", collectionDataHandler)

	m.Run()
}

package main

import (
	"demo/dbHelper"
	"demo/model"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func saveData(msgs <-chan amqp.Delivery) {
	dbHelper := dbHelper.NewDBHelper()
	dbHelper.Open("127.0.0.1", "cloundPlatform-mongod")
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

func main() {
	var rabbitmq = ""
	flag.StringVar(&rabbitmq, "Rabbitmq", "amqp://guest:guest@localhost:5672/", "rabbitmq address")
	flag.Parse()

	conn, err := amqp.Dial(rabbitmq)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when usused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
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

	forever := make(chan bool)

	go saveData(msgs)

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

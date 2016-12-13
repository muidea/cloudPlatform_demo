package main

import (
	"demo/model"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"

	"encoding/json"

	"github.com/streadway/amqp"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

// RtdRoutine 实时数据处理例程
type RtdRoutine struct {
	channel *amqp.Channel
}

// PostData 实时数据处理例程
func (rtd *RtdRoutine) PostData(request *model.RTDPacketRequest, response *model.RTDPacketResponse) error {
	q, err := rtd.channel.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	//log.Printf("PostData, dataSize:%d", len(request.RtdData))

	body, err := json.Marshal(request.RtdData)
	failOnError(err, "Failed to encode packet data")
	err = rtd.channel.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		})

	failOnError(err, "Failed to publish a message")

	return nil
}

func main() {
	var svrPort = 123400
	var rabbitmq = ""
	flag.IntVar(&svrPort, "Port", 0, "rtdService port")
	flag.StringVar(&rabbitmq, "Rabbitmq", "amqp://guest:guest@localhost:5672/", "rabbitmq address")
	flag.Parse()
	if svrPort == 0 {
		log.Print("illegal rtdService port.")
		return
	}

	forever := make(chan bool)

	conn, err := amqp.Dial(rabbitmq)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	rtdRoutline := &RtdRoutine{channel: ch}
	rpc.Register(rtdRoutline)
	rpc.HandleHTTP()

	strPort := fmt.Sprintf(":%d", svrPort)
	l, e := net.Listen("tcp", strPort)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
	log.Printf(" [*] To exit press CTRL+C")
	<-forever
}

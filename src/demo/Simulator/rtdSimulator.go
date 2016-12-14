package main

import (
	"demo/model"
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"time"
)

// Simulator 数据仿真器
type Simulator interface {
	Value() float32
}

const constLoop = 60

type impl struct {
	curValue  float32
	tickValue int
	forward   bool
}

func (s *impl) Value() float32 {
	curValue := s.curValue

	const step = 1.0
	s.tickValue++

	if s.forward {
		s.curValue += step
	} else {
		s.curValue -= step
	}

	if s.tickValue == constLoop {
		if s.forward {
			s.forward = false
		} else {
			s.forward = true
		}
		s.tickValue = 0
	}

	return curValue
}

// NewSimulator 新建仿真器
func NewSimulator() Simulator {
	i := impl{curValue: 20.0, tickValue: 0, forward: true}
	return &i
}

func simuValue(client *rpc.Client, simulatorName string, pointName []string) {
	point2Simulator := map[string]Simulator{}
	for _, point := range pointName {
		point2Simulator[point] = NewSimulator()
	}

	preTimeStamp := time.Now().UTC().UnixNano() / int64(time.Millisecond)
	args := &model.RTDPacketRequest{Sequence: 0}
	args.Source = simulatorName
	for true {
		args.RtdData = []model.RTDData{}
		args.Sequence++

		curTimeStamp := time.Now().UTC().UnixNano() / int64(time.Millisecond)
		for true {
			dif := curTimeStamp - preTimeStamp
			if dif < 1000 {
				time.Sleep(20 * time.Millisecond)

				curTimeStamp = time.Now().UTC().UnixNano() / int64(time.Millisecond)
				continue
			}

			break
		}
		preTimeStamp = curTimeStamp

		//log.Printf("post Data,timeStamp:%s", curTimeStamp.String())
		for k, v := range point2Simulator {
			rtd := model.RTDData{}
			rtd.Name = k
			rtd.Quality = 0
			rtd.Value = v.Value()
			rtd.TimeStamp = curTimeStamp
			args.RtdData = append(args.RtdData, rtd)
		}

		rsp := &model.RTDPacketResponse{}
		client.Go("RtdRoutine.PostData", args, rsp, nil)
	}
}

func main() {
	var svrIP = "127.0.0.1"
	var svrPort = 12340
	var simuName = ""
	var simueNum = 100
	var offSet = 100
	flag.IntVar(&svrPort, "Port", 0, "rtdService port")
	flag.StringVar(&svrIP, "IP", "", "rtdService ip")
	flag.StringVar(&simuName, "Name", "testSimulator", "simulator's name")
	flag.IntVar(&simueNum, "Number", 0, "point number")
	flag.IntVar(&offSet, "Offset", 0, "point name index offset")
	flag.Parse()
	if len(svrIP) == 0 || svrPort == 0 {
		log.Print("illegal rtdService address.")
		return
	}

	forever := make(chan bool)

	svrAddr := fmt.Sprintf("%s:%d", svrIP, svrPort)
	client, err := rpc.DialHTTP("tcp", svrAddr)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer client.Close()

	pointName := []string{}
	i := 1
	for true {
		vName := fmt.Sprintf("Point_%05d", offSet+i)
		pointName = append(pointName, vName)
		if i >= simueNum {
			break
		}

		i++
	}

	log.Printf("construct point ok, size:%d", len(pointName))

	go simuValue(client, simuName, pointName)

	log.Printf(" [*] To exit press CTRL+C")
	<-forever
}

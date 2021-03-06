package main

import (
	"demo/model"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/rpc"
	"time"
)

// Simulator 数据仿真器
type Simulator interface {
	Value() float32
}

type constImpl struct {
	curValue float32
}

func (s *constImpl) Value() float32 {
	return 50.0
}

type triangleImpl struct {
	curValue  float32
	tickValue int
	forward   bool
}

func (s *triangleImpl) Value() float32 {
	const constLoop = 60

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

type squareImpl struct {
	curValue  float32
	tickValue int
	forward   bool
}

func (s *squareImpl) Value() float32 {
	const highValue = 70.0
	const lowValue = 30.0
	const constScop = 20

	curValue := s.curValue

	s.tickValue++

	if s.forward {
		s.curValue = highValue
	} else {
		s.curValue = lowValue
	}

	if s.tickValue%constScop == 0 {
		if s.forward {
			s.forward = false
		} else {
			s.forward = true
		}
		s.tickValue = 0
	}

	return curValue
}

type randomImpl struct {
	curValue  float32
	tickValue int
	forward   bool
}

func (s *randomImpl) randFloat(min, max float32) float32 {
	if max-min <= 0 {
		return min
	}
	rand.Seed(time.Now().UTC().UnixNano())

	return min + float32(rand.Intn(int(max-min)))
}

func (s *randomImpl) Value() float32 {
	const highValue = 80.0
	const lowValue = 20.0

	curValue := s.randFloat(lowValue, highValue)

	return curValue
}

// NewSimulator 新建仿真器
func NewSimulator(dataType int) Simulator {
	switch dataType {
	case 0:
		i := triangleImpl{curValue: 20.0, tickValue: 0, forward: true}
		return &i
	case 1:
		i := squareImpl{curValue: 20.0, tickValue: 0, forward: true}
		return &i
	case 2:
		i := randomImpl{curValue: 20.0, tickValue: 0, forward: true}
		return &i
	default:
		i := constImpl{}
		return &i
	}
}

type rtdServer struct {
	svrAddr string
	client  *rpc.Client
}

func (s *rtdServer) Connect() bool {
	client, err := rpc.DialHTTP("tcp", s.svrAddr)
	if err != nil {
		log.Print("dialing:", err)
		return false
	}

	s.client = client
	return true
}

func (s *rtdServer) DisConnect() {
	s.client.Close()
	s.client = nil
}

func (s *rtdServer) Go(serviceMethod string, args interface{}, reply interface{}, done chan *rpc.Call) bool {
	if s.client == nil {
		if !s.Connect() {
			return false
		}
	}

	ret := s.client.Go(serviceMethod, args, reply, done)
	if ret != nil {
		if ret.Error != nil {
			s.DisConnect()

			return false
		}
	}

	return true
}

func newRtdRPCServer(rtdAddr string) *rtdServer {
	rtd := rtdServer{svrAddr: rtdAddr, client: nil}
	return &rtd
}

func simuValue(rtdSvr *rtdServer, simulatorName string, dataType int, pointName []string) {
	point2Simulator := map[string]Simulator{}
	for _, point := range pointName {
		point2Simulator[point] = NewSimulator(dataType)
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
		ret := rtdSvr.Go("RtdRoutine.PostData", args, rsp, nil)
		if !ret {
			log.Print("rpc call failed")
		}
	}
}

func main() {
	var svrIP = "127.0.0.1"
	var svrPort = 12340
	var simuName = ""
	var simueNum = 100
	var dataType = 0
	var offSet = 100
	flag.IntVar(&svrPort, "Port", 0, "rtdService port")
	flag.StringVar(&svrIP, "IP", "", "rtdService ip")
	flag.StringVar(&simuName, "Name", "testSimulator", "simulator's name")
	flag.IntVar(&simueNum, "Number", 0, "point number")
	flag.IntVar(&dataType, "DataType", 0, "simulate data type, 0:triangle, 1:square, 2:random, other: 50.0")
	flag.IntVar(&offSet, "Offset", 0, "point name index offset")
	flag.Parse()
	if len(svrIP) == 0 || svrPort == 0 {
		log.Print("illegal rtdService address.")
		return
	}

	forever := make(chan bool)

	svrAddr := fmt.Sprintf("%s:%d", svrIP, svrPort)
	rtdServer := newRtdRPCServer(svrAddr)
	defer rtdServer.DisConnect()

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

	go simuValue(rtdServer, simuName, dataType, pointName)

	log.Printf(" [*] To exit press CTRL+C")
	<-forever
}

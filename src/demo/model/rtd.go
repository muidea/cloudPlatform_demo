package model

// DataValue 数据值
type DataValue struct {
	Value     float32
	Quality   int32
	TimeStamp int64
}

// RTDData 实时数据
type RTDData struct {
	DataValue
	Name string
}

// RTDPacketRequest 实时数据请求
type RTDPacketRequest struct {
	Source   string
	Sequence int
	RtdData  []RTDData
}

// RTDPacketResponse 实时数据响应
type RTDPacketResponse struct {
	ErrCode int
	Reason  int
}

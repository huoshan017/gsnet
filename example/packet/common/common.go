package common

import (
	"bytes"
	"fmt"
)

const (
	connDataType = 0 // 连接数据类型
	TestAddress  = "127.0.0.1:9999"
)

type SendDataInfo struct {
	list [][]byte
	cnum int32
}

func CreateSendDataInfo(cnum int32) *SendDataInfo {
	return &SendDataInfo{
		list: make([][]byte, 0),
		cnum: cnum,
	}
}

// 发送goroutine中调用
func (info *SendDataInfo) AppendSendData(data []byte) {
	info.list = append(info.list, data)
}

// 在逻辑goroutine中调用
func (info *SendDataInfo) CompareData(data []byte, isForward bool) (bool, error) {
	if bytes.Equal(info.list[0], data) {
		if isForward {
			info.compareForward(false)
		}
		return true, nil
	}
	return false, fmt.Errorf("data %v compare info.list[0] %v failed", data, info.list[0])
}

func (info *SendDataInfo) compareForward(toLock bool) {
	info.list = info.list[1:]
}

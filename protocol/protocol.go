package protocol

import (
	"reflect"
	"sync"
)

// 协议数据
// 头和body 都属于协议数据
type ProtocolData interface {
	Decode(b []byte) error   // 从字节中读取
	Encode() ([]byte, error) // 编码
}

// 协议数据内容
type ProrocolBody = ProtocolData

var protocolDataPools = &typePools{
	pools: make(map[reflect.Type]*sync.Pool),
	New: func(t reflect.Type) interface{} {
		var argv reflect.Value

		if t.Kind() == reflect.Ptr { // reply must be ptr
			argv = reflect.New(t.Elem())
		} else {
			argv = reflect.New(t)
		}

		return argv.Interface()
	},
}

// 在使用的时候自己注意类型
func InitDataPool(t reflect.Type) {
	protocolDataPools.Init(t)
}

func PutTypeData(t reflect.Type, data ProtocolData) {
	protocolDataPools.Put(t, data)
}

func GetTypeData(t reflect.Type) ProtocolData {
	return protocolDataPools.Get(t).(ProtocolData)
}

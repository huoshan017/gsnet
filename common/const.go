package common

import "time"

const (
	DefaultSentAckTimeSpan            = time.Millisecond * 200 // 默认发送确认包间隔时间
	DefaultAckSentNum                 = 10                     // 默认确认发送包数
	DefaultHeartbeatTimeSpan          = time.Second * 10       // 默认发送心跳间隔时间
	DefaultMinimumHeartbeatTimeSpan   = time.Second * 3        // 最小心跳发送间隔
	DefaultDisconnectHeartbeatTimeout = time.Second * 30       // 断开连接的心跳超时
)

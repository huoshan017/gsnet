package common

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/huoshan017/gsnet/packet"
)

type GenCryptoKeyFunc func(*rand.Rand) []byte

// 选项结构
type Options struct {
	noDelay          bool
	keepAlived       bool
	keepAlivedPeriod time.Duration
	readTimeout      time.Duration
	writeTimeout     time.Duration
	tickSpan         time.Duration
	//dataProto                  IDataProto
	sendChanLen                int
	recvChanLen                int
	writeBuffSize              int
	readBuffSize               int
	connCloseWaitSecs          int                    // 連接關閉等待時間(秒)
	packetPool                 packet.IPacketPool     // 包池
	packetCompressType         packet.CompressType    // 包解压缩类型
	packetEncryptionType       packet.EncryptionType  // 包加解密类型
	genCryptoKeyFunc           GenCryptoKeyFunc       // 产生密钥的函数
	rand                       *rand.Rand             // 随机数
	createPacketHeaderFunc     CreatePacketHeaderFunc // 创建包头函数
	packetHeaderLength         uint8                  // 包头长度
	connDataType               int                    // 连接数据结构类型
	resendConfig               *ResendConfig          // 重发配置
	useHeartbeat               bool                   // 使用心跳
	heartbeatTimeSpan          time.Duration          // 心跳间隔
	minHeartbeatTimeSpan       time.Duration          // 最小心跳间隔
	disconnectHeartbeatTimeout time.Duration          // 断连的心跳超时
	customDatas                map[string]any         // 自定义数据

	// todo 以下是需要实现的配置逻辑
	// flushWriteInterval time.Duration // 写缓冲数据刷新到网络IO的最小时间间隔
}

// 创建Options
func NewOptions() *Options {
	return &Options{
		customDatas: make(map[string]any),
	}
}

// 选项
type Option func(*Options)

func (options *Options) GetCustomData(key string) any {
	return options.customDatas[key]
}

func (options *Options) SetCustomData(key string, data any) {
	options.customDatas[key] = data
}

func (options *Options) GetNodelay() bool {
	return options.noDelay
}

func (options *Options) SetNodelay(noDelay bool) {
	options.noDelay = noDelay
}

func (options *Options) GetKeepAlived() bool {
	return options.keepAlived
}

func (options *Options) SetKeepAlived(keepAlived bool) {
	options.keepAlived = keepAlived
}

func (options *Options) GetKeepAlivedPeriod() time.Duration {
	return options.keepAlivedPeriod
}

func (options *Options) SetKeepAlivedPeriod(keepAlivedPeriod time.Duration) {
	options.keepAlivedPeriod = keepAlivedPeriod
}

func (options *Options) GetReadTimeout() time.Duration {
	return options.readTimeout
}

func (options *Options) SetReadTimeout(timeout time.Duration) {
	options.readTimeout = timeout
}

func (options *Options) GetWriteTimeout() time.Duration {
	return options.writeTimeout
}

func (options *Options) SetWriteTimeout(timeout time.Duration) {
	options.writeTimeout = timeout
}

func (options *Options) GetTickSpan() time.Duration {
	return options.tickSpan
}

func (options *Options) SetTickSpan(span time.Duration) {
	options.tickSpan = span
}

func (options *Options) GetSendChanLen() int {
	return options.sendChanLen
}

func (options *Options) SetSendChanLen(chanLen int) {
	options.sendChanLen = chanLen
}

func (options *Options) GetRecvChanLen() int {
	return options.recvChanLen
}

func (options *Options) SetRecvChanLen(chanLen int) {
	options.recvChanLen = chanLen
}

func (options *Options) GetWriteBuffSize() int {
	return options.writeBuffSize
}

func (options *Options) SetWriteBuffSize(size int) {
	options.writeBuffSize = size
}

func (options *Options) GetReadBuffSize() int {
	return options.readBuffSize
}

func (options *Options) SetReadBuffSize(size int) {
	options.readBuffSize = size
}

func (options *Options) GetConnCloseWaitSecs() int {
	return options.connCloseWaitSecs
}

func (options *Options) SetConnCloseWaitSecs(secs int) {
	options.connCloseWaitSecs = secs
}

func (options *Options) GetPacketPool() packet.IPacketPool {
	return options.packetPool
}

func (options *Options) SetPacketPool(packetPool packet.IPacketPool) {
	options.packetPool = packetPool
}

func (options *Options) GetConnDataType() int {
	return options.connDataType
}

func (options *Options) SetConnDataType(typ int) {
	options.connDataType = typ
}

func (options *Options) GetResendConfig() *ResendConfig {
	return options.resendConfig
}

func (options *Options) SetResendConfig(config *ResendConfig) {
	if config.AckSentNum == 0 {
		config.AckSentNum = DefaultAckSentNum
	}
	if config.AckSentSpan == 0 {
		config.AckSentSpan = DefaultSentAckTimeSpan
	}
	options.resendConfig = config
}

func (options *Options) IsUseHeartbeat() bool {
	return options.useHeartbeat
}

func (options *Options) SetUseHeartbeat(use bool) {
	options.useHeartbeat = use
}

func (options *Options) GetHeartbeatTimeSpan() time.Duration {
	return options.heartbeatTimeSpan
}

func (options *Options) SetHeartbeatTimeSpan(span time.Duration) {
	options.heartbeatTimeSpan = span
}

func (options *Options) GetMinHeartbeatTimeSpan() time.Duration {
	return options.minHeartbeatTimeSpan
}

func (options *Options) SetMinHeartbeatTimeSpan(span time.Duration) {
	options.minHeartbeatTimeSpan = span
}

func (options *Options) GetDisconnectHeartbeatTimeout() time.Duration {
	return options.disconnectHeartbeatTimeout
}

func (options *Options) SetDisconnectHeartbeatTimeout(span time.Duration) {
	options.disconnectHeartbeatTimeout = span
}

func (options *Options) GetPacketCompressType() packet.CompressType {
	return options.packetCompressType
}

func (options *Options) SetPacketCompressType(ct packet.CompressType) {
	if !packet.IsValidCompressType(ct) {
		panic(fmt.Sprintf("compress type %v invalid", ct))
	}
	options.packetCompressType = ct
}

func (options *Options) GetPacketEncryptionType() packet.EncryptionType {
	return options.packetEncryptionType
}

func (options *Options) SetPacketEncryptionType(et packet.EncryptionType) {
	if !packet.IsValidEncryptionType(et) {
		panic(fmt.Sprintf("encryptioin type %v invalid", et))
	}
	options.packetEncryptionType = et
}

func (options *Options) GetGenCryptoKeyFunc() GenCryptoKeyFunc {
	return options.genCryptoKeyFunc
}

func (options *Options) SetGenCryptoKeyFunc(fun GenCryptoKeyFunc) {
	options.genCryptoKeyFunc = fun
}

func (options *Options) GetRand() *rand.Rand {
	if options.rand == nil {
		options.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return options.rand
}

func (options *Options) GetPacketHeaderLength() uint8 {
	return options.packetHeaderLength
}

func (options *Options) SetPacketHeaderLength(length uint8) {
	options.packetHeaderLength = length
}

func (options *Options) GetCreatePacketHeaderFunc() CreatePacketHeaderFunc {
	return options.createPacketHeaderFunc
}

func (options *Options) SetCreatePacketHeaderFunc(fun CreatePacketHeaderFunc) {
	options.createPacketHeaderFunc = fun
}

func WithNoDelay(noDelay bool) Option {
	return func(options *Options) {
		options.SetNodelay(noDelay)
	}
}

func WithKeepAlived(keepAlived bool) Option {
	return func(options *Options) {
		options.SetKeepAlived(keepAlived)
	}
}

func WithKeepAlivedPeriod(keepAlivedPeriod time.Duration) Option {
	return func(options *Options) {
		options.SetKeepAlivedPeriod(keepAlivedPeriod)
	}
}

func WithReadTimeout(timeout time.Duration) Option {
	return func(options *Options) {
		options.SetReadTimeout(timeout)
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(options *Options) {
		options.SetWriteTimeout(timeout)
	}
}

func WithTickSpan(span time.Duration) Option {
	return func(options *Options) {
		options.SetTickSpan(span)
	}
}

func WithSendChanLen(chanLen int) Option {
	return func(options *Options) {
		options.SetSendChanLen(chanLen)
	}
}

func WithRecvChanLen(chanLen int) Option {
	return func(options *Options) {
		options.SetRecvChanLen(chanLen)
	}
}

func WithWriteBuffSize(size int) Option {
	return func(options *Options) {
		options.SetWriteBuffSize(size)
	}
}

func WithReadBuffSize(size int) Option {
	return func(options *Options) {
		options.SetReadBuffSize(size)
	}
}

func WithConnCloseWaitSecs(secs int) Option {
	return func(options *Options) {
		options.SetConnCloseWaitSecs(secs)
	}
}

func WithPacketPool(packetPool packet.IPacketPool) Option {
	return func(options *Options) {
		options.SetPacketPool(packetPool)
	}
}

func WithConnDataType(typ int) Option {
	return func(options *Options) {
		options.SetConnDataType(typ)
	}
}

func WithResendConfig(config *ResendConfig) Option {
	return func(options *Options) {
		options.SetResendConfig(config)
	}
}

func WithUseHeartbeat(use bool) Option {
	return func(options *Options) {
		options.SetUseHeartbeat(use)
	}
}

func WithHeartbeatTimeSpan(span time.Duration) Option {
	return func(options *Options) {
		options.SetHeartbeatTimeSpan(span)
	}
}

func WithMinHeartbeatTimeSpan(span time.Duration) Option {
	return func(options *Options) {
		options.SetMinHeartbeatTimeSpan(span)
	}
}

func WithDisconnectHeartbeatTimeout(span time.Duration) Option {
	return func(options *Options) {
		options.SetDisconnectHeartbeatTimeout(span)
	}
}

func WithPacketCompressType(ct packet.CompressType) Option {
	return func(options *Options) {
		options.SetPacketCompressType(ct)
	}
}

func WithPacketEncryptionType(et packet.EncryptionType) Option {
	return func(options *Options) {
		options.SetPacketEncryptionType(et)
	}
}

func WithGenCryptoKeyFunc(fun GenCryptoKeyFunc) Option {
	return func(options *Options) {
		options.SetGenCryptoKeyFunc(fun)
	}
}

func WithPacketHeaderLength(length uint8) Option {
	return func(options *Options) {
		options.SetPacketHeaderLength(length)
	}
}

func WithCreatePacketHeaderFunc(fun CreatePacketHeaderFunc) Option {
	return func(options *Options) {
		options.SetCreatePacketHeaderFunc(fun)
	}
}

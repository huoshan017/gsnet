package options

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/huoshan017/gsnet/packet"
	kcp "github.com/huoshan017/kcpgo"
)

const (
	DefaultSentAckTimeSpan                  = time.Millisecond * 200 // 默认发送确认包间隔时间
	DefaultAckSentNum                       = 10                     // 默认确认发送包数量
	MaxAckSentNum                           = 100                    // 最大确认发送包数量
	DefaultHeartbeatTimeSpan                = time.Second * 10       // 默认发送心跳间隔时间
	DefaultMinimumHeartbeatTimeSpan         = time.Second * 3        // 最小心跳发送间隔
	DefaultDisconnectHeartbeatTimeout       = time.Second * 30       // 断开连接的心跳超时
	DefaultAutoReconnectSeconds       int32 = 5
)

type NetProto int32

const (
	NetProtoTCP  = iota
	NetProtoTCP4 = 1
	NetProtoTCP6 = 2
	NetProtoUDP  = 3
	NetProtoUDP4 = 4
	NetProtoUDP6 = 5
)

type GenCryptoKeyFunc func(*rand.Rand) []byte

type CreatePacketHeaderFunc func(*Options) packet.IPacketHeader

type ResendConfig struct {
	SentListLength int32
	AckSentSpan    time.Duration
	AckSentNum     int32
	UseLockFree    bool
}

// 选项结构
type Options struct {
	netProto                   NetProto
	noDelay                    bool
	keepAlived                 bool
	keepAlivedPeriod           time.Duration
	readTimeout                time.Duration
	writeTimeout               time.Duration
	tickSpan                   time.Duration
	sendListLen                int
	recvListLen                int
	sendListMode               int32                  // 发送队列模式
	writeBuffSize              int                    // 写缓冲大小
	readBuffSize               int                    // 讀缓冲大小
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
	autoReconnect              bool                   // 是否自动重连
	autoReconnectSeconds       int32                  // second
	kcpOptions                 *kcp.Options           // kcp options
}

// 创建Options
func NewOptions() *Options {
	return &Options{
		customDatas: make(map[string]any),
	}
}

// 选项
type Option func(*Options)

func (options *Options) GetNetProto() NetProto {
	return options.netProto
}

func (options *Options) SetNetProto(netProto NetProto) {
	options.netProto = netProto
}

func (options *Options) GetKcpOptions() *kcp.Options {
	return options.kcpOptions
}

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

func (options *Options) GetSendListLen() int {
	return options.sendListLen
}

func (options *Options) SetSendListLen(listLen int) {
	options.sendListLen = listLen
}

func (options *Options) GetRecvListLen() int {
	return options.recvListLen
}

func (options *Options) SetRecvListLen(listLen int) {
	options.recvListLen = listLen
}

func (options *Options) GetSendListMode() int32 {
	return options.sendListMode
}

func (options *Options) SetSendListMode(mode int32) {
	options.sendListMode = mode
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
	if config.AckSentNum <= 0 {
		config.AckSentNum = DefaultAckSentNum
	}
	if config.AckSentNum > MaxAckSentNum {
		config.AckSentNum = MaxAckSentNum
	}
	if config.AckSentSpan <= 0 {
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

func (options *Options) IsAutoReconnect() bool {
	return options.autoReconnect
}

func (options *Options) SetAutoReconnect(enable bool) {
	options.autoReconnect = enable
}

func (options *Options) GetAutoReconnectSeconds() int32 {
	secs := options.autoReconnectSeconds
	if secs <= 0 {
		secs = DefaultAutoReconnectSeconds
	}
	return secs
}

func (options *Options) SetAutoReconnectSeconds(secs int32) {
	options.autoReconnectSeconds = secs
}

func (options *Options) GetKcpStream() bool {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	return options.kcpOptions.GetStream()
}

func (options *Options) SetKcpStream(stream bool) {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	options.kcpOptions.SetStream(stream)
}

func (options *Options) GetKcpRTO() int32 {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	return options.kcpOptions.GetRTO()
}

func (options *Options) SetKcpRTO(rto int32) {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	options.kcpOptions.SetRTO(rto)
}

func (options *Options) GetKcpMinRTO() int32 {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	return options.kcpOptions.GetMinRTO()
}

func (options *Options) SetKcpMinRTO(rto int32) {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	options.kcpOptions.SetMinRTO(rto)
}

func (options *Options) GetKcpMtu() int32 {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	return options.kcpOptions.GetMtu()
}

func (options *Options) SetKcpMtu(mtu int32) {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	options.kcpOptions.SetMtu(mtu)
}

func (options *Options) GetKcpSendWnd() int32 {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	return options.kcpOptions.GetSendWnd()
}

func (options *Options) SetKcpSendWnd(wnd int32) {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	options.kcpOptions.SetSendWnd(wnd)
}

func (options *Options) GetKcpRecvWnd() int32 {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	return options.kcpOptions.GetRecvWnd()
}

func (options *Options) SetKcpRecvWnd(wnd int32) {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	options.kcpOptions.SetRecvWnd(wnd)
}

func (options *Options) GetKcpNodelay() int32 {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	return options.kcpOptions.GetNodelay()
}

func (options *Options) SetKcpNodelay(nodelay int32) {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	options.kcpOptions.SetNodelay(nodelay)
}

func (options *Options) SetKcpFastResend(resend int32) {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	options.kcpOptions.SetFastResend(resend)
}

func (options *Options) GetKcpNoCwnd() bool {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	return options.kcpOptions.GetNoCwnd()
}

func (options *Options) SetKcpNoCwnd(nocwnd bool) {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	options.kcpOptions.SetNoCwnd(nocwnd)
}

func (options *Options) GetKcpInterval() int32 {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	return options.kcpOptions.GetInterval()
}

func (options *Options) SetKcpInterval(interval int32) {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	options.kcpOptions.SetInterval(interval)
}

func (options *Options) GetKcpFastAckLimit() int32 {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	return options.kcpOptions.GetFastAckLimit()
}

func (options *Options) SetKcpFastAckLimit(limit int32) {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	options.kcpOptions.SetFastAckLimit(limit)
}

func (options *Options) GetKcpSSThresh() int32 {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	return options.kcpOptions.GetSSThresh()
}

func (options *Options) SetKcpSSThresh(ssthresh int32) {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	options.kcpOptions.SetSSThresh(ssthresh)
}

func (options *Options) GetKcpDeadLink() int32 {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	return options.kcpOptions.GetDeadLink()
}

func (options *Options) SetKcpDeadLink(deadlink int32) {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	options.kcpOptions.SetDeadLink(deadlink)
}

func (options *Options) GetKcpUserFreeOutputBuf() bool {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	return options.kcpOptions.IsUserFreeOutputBuf()
}

func (options *Options) SetKcpUserFreeOutputBuf(userfree bool) {
	if options.kcpOptions == nil {
		options.kcpOptions = &kcp.Options{}
	}
	options.kcpOptions.SetUserFreeOutputBuf(userfree)
}

func WithNetProto(netProto NetProto) Option {
	return func(options *Options) {
		options.SetNetProto(netProto)
	}
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

func WithSendListLen(listLen int) Option {
	return func(options *Options) {
		options.SetSendListLen(listLen)
	}
}

func WithRecvListLen(listLen int) Option {
	return func(options *Options) {
		options.SetRecvListLen(listLen)
	}
}

func WithSendListMode(mode int32) Option {
	return func(options *Options) {
		options.SetSendListMode(mode)
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

func WithAutoReconnect(enable bool) Option {
	return func(options *Options) {
		options.SetAutoReconnect(enable)
	}
}

func WithAutoReconnectSeconds(secs int32) Option {
	return func(options *Options) {
		options.SetAutoReconnectSeconds(secs)
	}
}

// -- kcp options
func WithKcpStream(stream bool) Option {
	return func(options *Options) {
		options.SetKcpStream(stream)
	}
}

func WithKcpMtu(mtu int32) Option {
	return func(options *Options) {
		options.SetKcpMtu(mtu)
	}
}

func WithKcpMinRTO(rto int32) Option {
	return func(options *Options) {
		options.SetKcpMinRTO(rto)
	}
}

func WithKcpRTO(rto int32) Option {
	return func(options *Options) {
		options.SetKcpRTO(rto)
	}
}

func WithKcpSendWnd(wnd int32) Option {
	return func(options *Options) {
		options.SetKcpSendWnd(wnd)
	}
}

func WithKcpRecvWnd(wnd int32) Option {
	return func(options *Options) {
		options.SetKcpRecvWnd(wnd)
	}
}

func WithKcpNodelay(nodelay int32) Option {
	return func(options *Options) {
		options.SetKcpNodelay(nodelay)
	}
}

func WithKcpFastResend(resend int32) Option {
	return func(options *Options) {
		options.SetKcpFastResend(resend)
	}
}

func WithKcpNoCwnd(nocwnd bool) Option {
	return func(options *Options) {
		options.SetKcpNoCwnd(nocwnd)
	}
}

func WithKcpInterval(interval int32) Option {
	return func(options *Options) {
		options.SetKcpInterval(interval)
	}
}

func WithKcpSSThresh(ssthresh int32) Option {
	return func(options *Options) {
		options.SetKcpSSThresh(ssthresh)
	}
}

func WithKcpFastAckLimit(limit int32) Option {
	return func(options *Options) {
		options.SetKcpFastAckLimit(limit)
	}
}

func WithKcpDeadLink(deadlink int32) Option {
	return func(options *Options) {
		options.SetKcpDeadLink(deadlink)
	}
}

func WithKcpUserFreeOutputBuf(userfree bool) Option {
	return func(options *Options) {
		options.SetKcpUserFreeOutputBuf(userfree)
	}
}

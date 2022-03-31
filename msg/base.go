package msg

type MsgIdType uint32

type IMsgCodec interface {
	Encode(interface{}) ([]byte, error)
	Decode([]byte, interface{}) error
}

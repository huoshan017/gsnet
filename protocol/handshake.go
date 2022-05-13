package protocol

// client syn
// server ack

type HandshakeDataType int32

const (
	HandshakeDataTypeNone           HandshakeDataType = iota
	HandshakeDataTypeCompressType   HandshakeDataType = 1
	HandshakeDataTypeEncryptionType HandshakeDataType = 2
	HandshakeDataTypeEncryptionKey  HandshakeDataType = 3
	HandshakeDataTypeSessionId      HandshakeDataType = 4
	HandshakeDataTypeSessionKey     HandshakeDataType = 5
)

type HandshakeCodec interface {
	Encode(map[int32]any) []byte
	Decode([]byte) (map[int32]any, error)
}

type DefaultHandshakeCodec struct{}

func (c DefaultHandshakeCodec) Encode(m map[HandshakeDataType]any) []byte {
	return nil
}

func (c DefaultHandshakeCodec) Decode(data []byte) (map[HandshakeDataType]any, error) {
	return nil, nil
}

package msg

/*
type IMsgHeader interface {
	SetId(id MsgIdType)
	GetId() MsgIdType
	Set(string, any)
	Get(string) any
	FormatTo([]byte) error
	UnformatFrom([]byte) error
}

type CreateMsgHeaderFunc func(options *MsgOptions) IMsgHeader

type DefaultMsgHeader struct {
	id MsgIdType
}

func (h *DefaultMsgHeader) SetId(id MsgIdType) {
	h.id = id
}

func (h *DefaultMsgHeader) GetId() MsgIdType {
	return h.id
}

func (h *DefaultMsgHeader) Set(string, any) {
}

func (h *DefaultMsgHeader) Get(string) any {
	return nil
}

func (h *DefaultMsgHeader) FormatTo(buf []byte) error {
	if len(buf) < DefaultMsgHeaderLength {
		return ErrMsgHeaderFormatNeedLargerBuffer
	}
	for i := 0; i < DefaultMsgHeaderLength; i++ {
		buf[i] = byte(h.id >> (8 * (DefaultMsgHeaderLength - i - 1)))
	}
	return nil
}

func (h *DefaultMsgHeader) UnformatFrom(buf []byte) error {
	if len(buf) < DefaultMsgHeaderLength {
		return ErrMsgBodyIncomplete
	}
	for i := 0; i < DefaultMsgHeaderLength; i++ {
		h.id += MsgIdType(buf[i] << (8 * (DefaultMsgHeaderLength - i - 1)))
	}
	return nil
}

func NewDefaultMsgHeader(options *MsgOptions) IMsgHeader {
	return &DefaultMsgHeader{}
}
*/

func DefaultMsgHeaderFormat(msgid MsgIdType, data []byte) error {
	if len(data) < DefaultMsgHeaderLength {
		return ErrMsgHeaderFormatNeedLargerBuffer
	}
	for i := 0; i < DefaultMsgHeaderLength; i++ {
		data[i] = byte(msgid >> (8 * (DefaultMsgHeaderLength - i - 1)))
	}
	return nil
}

func DefaultMsgHeaderUnformat(data []byte) (MsgIdType, error) {
	if len(data) < DefaultMsgHeaderLength {
		return 0, ErrMsgBodyIncomplete
	}
	var msgid MsgIdType
	for i := 0; i < DefaultMsgHeaderLength; i++ {
		msgid += (MsgIdType(data[i]) << (8 * (DefaultMsgHeaderLength - i - 1)))
	}
	return msgid, nil
}

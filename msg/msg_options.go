package msg

import (
	"unsafe"

	"github.com/huoshan017/gsnet/options"
)

type MsgHeaderFormatFunc func(MsgIdType, []byte) error
type MsgHeaderUnformatFunc func([]byte) (MsgIdType, error)

type MsgOptions struct {
	headerLength uint8
	formatFunc   MsgHeaderFormatFunc
	unformatFunc MsgHeaderUnformatFunc
	customDatas  map[string]any
}

func (options *MsgOptions) SetHeaderLength(length uint8) {
	options.headerLength = length
}

func (options *MsgOptions) GetHeaderLength() uint8 {
	return options.headerLength
}

func (options *MsgOptions) SetHeaderFormatFunc(fn MsgHeaderFormatFunc) {
	options.formatFunc = fn
}

func (options *MsgOptions) GetHeaderFormatFunc() MsgHeaderFormatFunc {
	return options.formatFunc
}

func (options *MsgOptions) SetHeaderUnformatFunc(fn MsgHeaderUnformatFunc) {
	options.unformatFunc = fn
}

func (options *MsgOptions) GetHeaderUnformatFunc() MsgHeaderUnformatFunc {
	return options.unformatFunc
}

func (options *MsgOptions) SetCustomData(key string, data any) {
	if options.customDatas == nil {
		options.customDatas = make(map[string]any)
	}
	options.customDatas[key] = data
}

func (options *MsgOptions) GetCustomData(key string) any {
	if options.customDatas == nil {
		return nil
	}
	return options.customDatas[key]
}

type MsgClientOptions struct {
	options.ClientOptions
	MsgOptions
}

type MsgServerOptions struct {
	options.ServerOptions
	MsgOptions
}

func WithHeaderLength(length uint8) options.Option {
	return func(options *options.Options) {
		withMsgOptionValue(options, func(mcp *MsgClientOptions, msp *MsgServerOptions) {
			if mcp != nil {
				mcp.SetHeaderLength(length)
			} else if msp != nil {
				msp.SetHeaderLength(length)
			}
		})
	}
}

func WithHeaderFormatFunc(fn MsgHeaderFormatFunc) options.Option {
	return func(options *options.Options) {
		withMsgOptionValue(options, func(mcp *MsgClientOptions, msp *MsgServerOptions) {
			if mcp != nil {
				mcp.SetHeaderFormatFunc(fn)
			} else if msp != nil {
				msp.SetHeaderFormatFunc(fn)
			}
		})
	}
}

func WithHeaderUnformatFunc(fn MsgHeaderUnformatFunc) options.Option {
	return func(options *options.Options) {
		withMsgOptionValue(options, func(mcp *MsgClientOptions, msp *MsgServerOptions) {
			if mcp != nil {
				mcp.SetHeaderUnformatFunc(fn)
			} else if msp != nil {
				msp.SetHeaderUnformatFunc(fn)
			}
		})
	}
}

func withMsgOptionValue(ops *options.Options, setValue func(*MsgClientOptions, *MsgServerOptions)) {
	cp := (*options.ClientOptions)(unsafe.Pointer(ops))
	if cp != nil {
		mcp := (*MsgClientOptions)(unsafe.Pointer(cp))
		if mcp == nil {
			panic("type *Options transfer to *MsgClientOptions failed")
		}
		setValue(mcp, nil)
	} else {
		sp := (*options.ServerOptions)(unsafe.Pointer(ops))
		if sp == nil {
			msp := (*MsgServerOptions)(unsafe.Pointer(sp))
			if msp == nil {
				panic("type *Options transfer to *MsgServerOptions failed")
			}
			setValue(nil, msp)
		}
	}
}

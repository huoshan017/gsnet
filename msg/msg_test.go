package msg

import (
	"testing"

	"github.com/huoshan017/gsnet/common"
)

type testData4ServerOptions struct {
	options MsgServerOptions
}

type testData4ClientOptions struct {
	options MsgClientOptions
}

func (d *testData4ServerOptions) check(options ...common.Option) {
	for i := 0; i < len(options); i++ {
		options[i](&d.options.Options)
	}
}

func (d *testData4ClientOptions) check(options ...common.Option) {
	for i := 0; i < len(options); i++ {
		options[i](&d.options.Options)
	}
}

func TestMsgOptions(t *testing.T) {
	var ds = testData4ServerOptions{}
	ds.check(WithHeaderLength(4))

	var dc = testData4ClientOptions{}
	dc.check(WithHeaderLength(10))
}

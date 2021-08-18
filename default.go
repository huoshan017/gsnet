package gsnet

import "time"

type DefaultServiceCallback struct {
}

func (c *DefaultServiceCallback) OnConnect(sid uint64) {
}

func (c *DefaultServiceCallback) OnDisconnect(sid uint64, err error) {
}

func (c *DefaultServiceCallback) OnError(err error) {
}

func (c *DefaultServiceCallback) OnTick(tick time.Duration) {
}

type DefaultClientCallback struct {
}

func (c *DefaultClientCallback) OnConnect() {
}

func (c *DefaultClientCallback) OnDisconnect() {
}

func (c *DefaultClientCallback) OnError(err error) {
}

func (c *DefaultClientCallback) OnTick(tick time.Duration) {
}

package common

import (
	"sync"
	"testing"
)

type ITest interface {
	Logf(string, ...any)
	Errorf(string, ...any)
}

func testSendList(t ITest, sendList ISendList) {
	var (
		num int = 100000000
		wg  sync.WaitGroup
	)

	defer func() {
		sendList.Finalize()
	}()

	if b, o := t.(*testing.B); o {
		num = b.N
	}

	wg.Add(num)

	// read goroutine
	go func() {
		i := 0
		for ; i < num; i++ {
			sendList.PopFront()
			wg.Done()
		}
		t.Logf("read %v num", i)
	}()

	// write goroutine
	go func() {
		i := 0
		for ; i < num; i++ {
			if err := sendList.PushBack(nullWrapperSendData); err != nil {
				t.Errorf("num i %v, unlimited channel err %v", i, err)
			}
		}
		t.Logf("write %v num", i)
	}()

	wg.Wait()

	sendList.Close()
}

func TestCSendList(t *testing.T) {
	testSendList(t, newCondSendList())
}

func TestUChan(t *testing.T) {
	testSendList(t, newUnlimitedChan())
}

func BenchmarkCSendList(b *testing.B) {
	testSendList(b, newCondSendList())
}

func BenchmarkUChan(b *testing.B) {
	testSendList(b, newUnlimitedChan())
}

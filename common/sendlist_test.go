package common

import (
	"sync"
	"testing"
)

func testSendList(t *testing.T, sendList ISendList) {
	var (
		num int = 100000000
		wg  sync.WaitGroup
	)

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
			if !sendList.PushBack(nullWrapperSendData) {
				t.Errorf("num i %v, unlimited channel closed", i)
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

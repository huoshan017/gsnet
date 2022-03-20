package test

import (
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	ts := createTestServer(t, 2)
	err := ts.Listen(testAddress)
	if err != nil {
		t.Errorf("server for test client listen err: %+v", err)
		return
	}
	defer ts.End()

	go ts.Start()

	t.Logf("server for test client running")

	tc := createTestClient(t, 1)
	err = tc.Connect(testAddress)
	if err != nil {
		t.Errorf("test client connect err: %+v", err)
		return
	}
	go tc.Run()

	t.Logf("test client running")

	for i := 0; i < 10000; i++ {
		err := tc.Send([]byte("abcdefghijklmnopqrstuvwxyz0123456789"))
		if err != nil {
			t.Errorf("test client send err: %+v", err)
			return
		}
		time.Sleep(time.Millisecond)
	}

	t.Logf("test done")
}

func BenchmarkClient(b *testing.B) {
	bs := createBenchmarkServerWithHandler(b, 1)
	err := bs.Listen(testAddress)
	if err != nil {
		b.Errorf("server for benchmark client listen err: %+v", err)
		return
	}
	defer bs.End()

	go bs.Start()

	b.Logf("server for benchmark client running")

	bc := createBenchmarkClient(b, 1)
	err = bc.Connect(testAddress)
	if err != nil {
		b.Errorf("benchmark client connect err: %+v", err)
		return
	}
	go bc.Run()

	b.Logf("benchmark client running")

	for i := 0; i < 10000; i++ {
		err := bc.Send([]byte("abcdefghijklmnopqrstuvwxyz0123456789"))
		if err != nil {
			b.Errorf("benchmark client send err: %+v", err)
			return
		}
		time.Sleep(time.Millisecond)
	}

	b.Logf("benchmark done")
}

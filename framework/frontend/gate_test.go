package frontend

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestRing(t *testing.T) {
	var (
		rand    = rand.New(rand.NewSource(time.Now().UnixNano()))
		ring    = newRing()
		idCache []int32
	)
	for i := 0; i < 100; i++ {
		n1 := rand.Int31n(5000) + 1
		for k := int32(0); k < n1; k++ {
			id := rand.Int31n(1000) + 1
			if !ring.write(id) {
				break
			}
			idCache = append(idCache, id)
		}
		n2 := rand.Int31n(6000) + 1
		for j := int32(0); j < n2; j++ {
			id, o := ring.peek()
			if !o {
				break
			}
			if idCache[0] != id {
				panic(fmt.Sprintf("compare idCache[0] %v to %v failed, j = %v, n1 = %v, n2 = %v", idCache[0], id, j, n1, n2))
			}
			idCache = idCache[1:]
			ring.advance()
		}
	}
}

var letters = []byte("abcdefghijklmnopqrstuvwxyz01234567890~!@#$%^&*()_+-={}[]|:;'<>?/.,")
var lettersLen = len(letters)

func randBytes(n int, ran *rand.Rand) []byte {
	b := make([]byte, n)
	for i := 0; i < len(b); i++ {
		r := ran.Int31n(int32(lettersLen))
		b[i] = letters[r]
	}
	return b
}

func TestAgentDataListMap(t *testing.T) {
	var (
		rand        = rand.New(rand.NewSource(time.Now().UnixNano()))
		idDataCache []*struct {
			data []byte
			id   int32
		}
	)
	idList := newRing()
	listMap := newAgentDataListMap()
	for i := 0; i < 300; i++ {
		// 这是发送
		n1 := rand.Int31n(500)
		for m1 := int32(0); m1 < n1; m1++ {
			// 先模拟记录发送id
			id := rand.Int31n(100) + 1
			if !idList.write(id) {
				break
			}

			data := randBytes(128, rand)
			listMap.pushBack(id, data)
			idDataCache = append(idDataCache, &struct {
				data []byte
				id   int32
			}{data, id})
		}

		// 这是接收
		n2 := rand.Int31n(300)
		for m2 := int32(0); m2 < n2; m2++ {
			id, o := idList.peek()
			if !o {
				break
			}
			var data []byte
			data, o = listMap.popFront(id)
			if !o {
				panic(fmt.Sprintf("id %v data cant pop", id))
			}
			if idDataCache[0].id != id || !bytes.Equal(idDataCache[0].data, data) {
				panic(fmt.Sprintf("id: %v, data %v;\r\nidDataCache[0]: id %v, data %v", id, data, idDataCache[0].id, idDataCache[0].data))
			}
			idDataCache = idDataCache[1:]
			idList.advance()
		}
	}
}

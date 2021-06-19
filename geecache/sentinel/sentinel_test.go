package sentinel

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
)

var (
	peers = []string {
	"http://localhost:8001",
	"http://localhost:8002",
	"http://localhost:8003",
	}
)

func Handle (w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write([]byte("hello"))
}

func startHttpServer()  {
	http.HandleFunc("/_geecache/", Handle)
	for i := 1; i <= 3; i++ {
		go http.ListenAndServe("127.0.0.1:800" + fmt.Sprint(i), nil)
	}
}

func TestHeartBeating(t *testing.T) {
	go startHttpServer()

	s := NewSentinel("http://localhost:10000", peers, 0, 0)
	s.HeartBeating()
	time.Sleep(10 * time.Second)
}

func TestHandleFailMsg(t *testing.T) {
	go startHttpServer()

	s := NewSentinel("http://localhost:10000", peers, 0, 0)
	go s.HeartBeating()
	s.HandleFailMsg()
	//假如停掉8003的端口
	if !reflect.DeepEqual(s.peers, []string{
		"http://localhost:8001",
		"http://localhost:8002",
	}) {
		t.Fail()
	}
	//再次停掉8001和8002端口
	if s.peers != nil {
		t.Fatal()
	}
}

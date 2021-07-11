package sentinel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"time"
)

var (
	defaultPingCount       = 4
	defaultPingSize        = 32
	defaultPingTime        = 1000
	defaultRequestInterval = time.Second * 3
	defaultPath            = "/_geecache"
	defaultPUTPath         = "/sentinel"
)

type HTTPSentinel struct {
	self            string
	//waitTime        time.Duration
	requestInterval time.Duration
	peers			map[string]bool
	ch              chan failMsg
}

type failMsg struct {
	DetectedTime time.Time 	`json:"detected_time"`
	SentinelName string 	`json:"sentinel_name"`
	PeerName string			`json:"peer_name"`
}

func (S *HTTPSentinel) Log(format string, v ...interface{}) {
	log.Printf("[Sentinel %s] %s", S.self, fmt.Sprintf(format, v...))
}

func NewSentinel (self string, peers []string, waitTime time.Duration, requestInterval time.Duration) *HTTPSentinel {
	sentinel := &HTTPSentinel{
		self:            self,
		//waitTime:        waitTime,
		requestInterval: requestInterval,
		peers: make(map[string]bool, len(peers)),
		ch: make(chan failMsg, len(peers) * 3),
	}

	for _, v := range peers {
		sentinel.peers[v] = false
	}

	//if sentinel.waitTime == 0 {
	//	sentinel.waitTime = defaultPingTime
	//}

	if sentinel.requestInterval == 0 {
		sentinel.requestInterval = defaultRequestInterval
	}

	return sentinel
}

func (S *HTTPSentinel) HeartBeating() {
	for k, _ := range S.peers {
		go S.RecvHttpMsg(k)
	}
}

func (S *HTTPSentinel) RecvHttpMsg(peer string) {
	ticker := time.NewTicker(S.requestInterval)
	op := NewPingOption(defaultPingCount, defaultPingSize, int64(defaultPingTime))
	//使用正则表达式提取Addr
	r := regexp.MustCompile("[a-zA-Z0-9]+")
	//割后变成[http localhost 8001]
	peerAddrs := r.FindAllString(peer, -1)
	//选择localhost
	peerAddr := peerAddrs[1]

	for range ticker.C {
		t := 0
		for i := 0; i < 3; i++ {
			if res := Ping(peerAddr, op); res == false {
				t++
			}
		}
		if  t >= 2 {
			S.peers[peer] = false

			failMsg := failMsg{
				DetectedTime: time.Now(),
				PeerName: peer,
				SentinelName: S.self,
			}
			S.ch <- failMsg

			S.Log("%s %s","connect failed with", peer)
			ticker.Stop()

		} else {
			S.Log("%s %s", "connect successfully with", peer)
			S.peers[peer] = true
		}
	}
}

func (S *HTTPSentinel) HandleFailMsg() {
	for  {
		msg := <-S.ch
		S.Log("%s %s", "handle failed peer", msg.PeerName)
		for peer, status := range S.peers {
			if status == true {
				go S.SendFailPeer(peer, msg)
			}
		}
	}
}

func (S *HTTPSentinel) SendFailPeer(peer string, Msg failMsg) {
	url := peer + defaultPUTPath

	body, _ := json.Marshal(Msg)
	buf := bytes.NewReader(body)

	req, _ := http.NewRequest("PUT", url, buf)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		S.Log("%s %s %v", "send new peers failed to", url, err)
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		S.Log("%v", err)
	}
	S.Log("%s", string(body))
}

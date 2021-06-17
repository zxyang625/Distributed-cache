package sentinel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	defaultWaitTime        = time.Second * 3
	defaultRequestInterval = time.Second * 2
	defaultPath            = "/_geecache"
	defaultPUTPath         = "/sentinel"
	mu 						sync.RWMutex
)

type HTTPSentinel struct {
	self            string
	waitTime        time.Duration
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
		waitTime:        waitTime,
		requestInterval: requestInterval,
		peers: make(map[string]bool, len(peers)),
		ch: make(chan failMsg, len(peers) * 3),
	}
	for _, v := range peers{
		sentinel.peers[v] = false
	}
	if sentinel.waitTime == 0 {
		sentinel.waitTime = defaultWaitTime
	}
	if sentinel.requestInterval == 0 {
		sentinel.requestInterval = defaultRequestInterval
	}
	return sentinel
}

func (S *HTTPSentinel) HeartBeating() {
	for peer, status := range S.peers {
		go S.RecvHttpMsg(peer, status)
	}
}

func (S *HTTPSentinel) RecvHttpMsg(peer string, status bool) {
	ticker := time.NewTicker(S.requestInterval)

	for range ticker.C {
		client := http.Client{
			Timeout: S.waitTime,
		}
		resp, err := client.Get(peer + defaultPath)

		if err != nil {
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
			bytes, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				S.Log("%s %s", peer, string(bytes))
				S.peers[peer] = true
				resp.Body.Close()
			}
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
	_, err := http.DefaultClient.Do(req)
	if err != nil {
		S.Log("%s %s %v", "send new peers message failed to", url, err)
	}

	S.Log("%s %s", "send new peers message succeed to", peer)
}

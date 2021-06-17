package geecache

import (
	"cache/geecache/consistenthash"
	"cache/geecache/pb"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const defaultBasePath = "/_geecache/"
const defaultReplicas = 3

// HTTPPool 作为承载结点间HTTP通信的核心数据结构
type HTTPPool struct {
	self string			//自己的地址,包括ip和端口
	basePath string		//basePath，作为节点间通讯地址的前缀
	mu sync.Mutex
	peers *consistenthash.Map	//用来根据具体的 key 选择节点
	httpGetters map[string]*httpGetter	//映射远程节点与对应的 httpGetter。每一个远程节点对应一个 httpGetter，因为 httpGetter 与远程节点的地址 baseURL 有关。
}

type failMsg struct {
	DetectedTime time.Time 	`json:"detected_time"`
	SentinelName string 	`json:"sentinel_name"`
	PeerName string			`json:"peer_name"`
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self: self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

func (p *HTTPPool) ListenSentinel(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	msg := failMsg{}
	json.Unmarshal(body, &msg)

	delete(p.httpGetters, msg.PeerName)
	p.peers.Remove(msg.PeerName)
	//msg.detectedTime = time.Now()
	//resp, _ := json.Marshal(msg)
	//w.Header().Set("Content-Type", "application/json")
	//w.Write(resp)
	fmt.Println(p.self, "delete succeed")
}

func (p *HTTPPool) ResponseStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write([]byte("hello"))
}

func (p *HTTPPool) GetKey(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(r.URL.Path[len(defaultBasePath):], "/", 2)
	groupName := parts[0]
	key := parts[1]
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: " + groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//使用gRPC通信
	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	//w.Write(view.ByteSlice())
	w.Write(body)
}

func (p *HTTPPool) Set(peers ...string) {
	fmt.Println("func (p *HTTPPool) Set(peers ...string)")
	p.mu.Lock()
	defer p.mu.Unlock()

	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
		//peer 类似:http://localhost:8001
		//p.basePath类似:/_geecache/
		//fmt.Printf("peer:%s p.basePath:%s\n", peer, p.basePath)
	}
}

func (p *HTTPPool) PickPeer (key string) (PeerGetter, bool) {
	fmt.Println("func (p *HTTPPool) PickPeer (key string) (PeerGetter, bool)")
	p.mu.Lock()
	defer p.mu.Unlock()

	if peer := p.peers.Get(key); peer != "" && peer != p.self{
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	} else  {
		p.Log("Peer pick self %s, search DB", peer)
		return nil, false
	}
}

//httpGetter 是客户端类,实现PeerGetter接口
type httpGetter struct {
	baseURL string	//baseURL 表示将要访问的远程节点的地址，例如 http://example.com/_geecache/
}

// 未使用gRPC的Get方法
//func (h *httpGetter) Get(group string, key string) ([]byte, error) {
//	u := fmt.Sprintf("%v%v/%v",
//		h.baseURL,
//		url.QueryEscape(group),
//		url.QueryEscape(key),
//		)
//
//	res, err := http.Get(u)
//	if err != nil {
//		return nil, err
//	}
//	defer res.Body.Close()
//
//	if res.StatusCode != http.StatusOK {
//		return nil, fmt.Errorf("server returned: %v", res.Status)
//	}
//
//	bytes, err := ioutil.ReadAll(res.Body)
//	if err != nil {
//		return nil, fmt.Errorf("reading response body: %v", err)
//	}
//
//	return bytes, nil
//}

//使用了gRPC的Get方法
func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	fmt.Println("func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error ")
	u := fmt.Sprintf("%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
		)

	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}


var _ PeerGetter = (*httpGetter)(nil)

var _ PeerPicker = (*HTTPPool)(nil)

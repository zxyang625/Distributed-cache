package main

import (
	"cache/geecache"
	"flag"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom" : "630",
	"Jack" : "589",
	"Sam" : "567",
}

func createGroup() *geecache.Group {
	return geecache.NewGroup("scores", 2 << 10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

//startCacheServer 用来启动缓存服务器，创建HTTPPool，添加结点信息，注册到gee中,
//启动http服务,一共三个端口，用户不感知
func startCacheServer(addr string, addrs []string, gee *geecache.Group) {
	peers := geecache.NewHTTPPool(addr)
	//对每一个结点都要告知其他结点的地址
	peers.Set(addrs...)
	gee.RegisterPeers(peers)
	log.Println("geecache is running at:", addr)
	//fmt.Println("addr[7:]:", addr[7:])	//例如:localhost:8001
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

//startAPIServer 用来启动API服务，与用户进行交互，用户感知
func startAPIServer(apiAddr string, gee *geecache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		}))
	log.Println("fontend server is running at", apiAddr)
	//fmt.Println("apiAddr[7:]", apiAddr[7:])	//localhost:9999
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {
	/*一致性哈希测试
	geecache.NewGroup("scores", 2 << 10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	addr := "localhost:9999"
	peers := geecache.NewHTTPPool(addr)
	log.Println("geecache is running at ", addr)
	log.Fatal(http.ListenAndServe(addr, peers))*/

	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroup()
	if api {
		go startAPIServer(apiAddr, gee)
	}

	startCacheServer(addrMap[port], addrs, gee)
}

package geecache

import "cache/geecache/pb"

//PeerGetter 接口中的Get()方法用于从对应group查找缓存值， PeerGetter 就对应于下面的HTTP客户端
type PeerGetter interface {
	//Get(group string, key string) ([]byte, error)
	Get(in *pb.Request, out *pb.Response) error
}

//PeerPicker 方法用于根据传入的key选择相应结点peer
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}


package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

type Map struct {
	hash Hash				//hash函数
	replicas int			//虚拟节点倍数
	keys []int				//哈希环
	hashMap map[int]string	//虚拟节点和真实结点的映射,键是虚拟节点的哈希值，值是真实节点的名称。
}

func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash: fn,
		hashMap: make(map[int]string),
	}
	//hash算法默认采用crc32/ChecksumIEEE算法
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

//Add 方法对应的是增加Map的真实结点，传入的是若干个结点的名称
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {	//对每个结点,创建replicas个虚拟节点,虚拟节点的名字是[编号]+真实结点的名字
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))	//计算每个虚拟结点的hash值
			m.keys = append(m.keys, hash)						//将对应的哈希值添加到环上
			m.hashMap[hash] = key								//建立虚拟结点hash值和名字之间的映射
		}
	}

	sort.Ints(m.keys)											//最后对所有哈希值排序
}

// Get 方法实现选择结点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	idx := sort.Search(len(m.keys), func(i int) bool {			//二分查找key在keys中满足条件的虚拟节点
		return m.keys[i] >= hash
	})

	//keys通过索引获取hash值，hashMap通过hash值获取真实结点。当key比m.keys所有都大的时候，idx可能等于len(m.keys)所以需要取余
	return m.hashMap[m.keys[idx % len(m.keys)]]
}
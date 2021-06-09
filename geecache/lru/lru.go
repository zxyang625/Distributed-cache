package lru

import "container/list"

type Cache struct {
	//允许的最大内存
	maxBytes int64
	//当前使用内存
	nbytes int64
	ll *list.List
	cache map[string]*list.Element
	//某条记录被移除的回调函数，可以是nil
	OnEvicted func(key string, value Value)
}

//entry 是双线链表结点的数据类型，保存key是便于在删除首节点时可以找到key
type entry struct {
	key string
	value Value
}

//Value 接口只有一个 Len 函数是为了保证通用性，即实现了 Value 接口的任意类型
type Value interface {
	//Len 函数返回值所占的内存大小
	Len() int
}

//New 函数用于实例化 Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes: maxBytes,
		ll: list.New(),
		cache: make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

//Get 实现从字典中找到对应的双向链表的结点，然后将其移动到队首
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// RemoveOldest 移除最近最少访问的结点
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Add 函数用于增加/修改
// 如果key存在则更新对应结点的值，并将该结点移到队首; 不存在则代表新结点，向字典中添加
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Len 获取数据条数
func (c *Cache) Len() int {
	return c.ll.Len()
}
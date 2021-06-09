package singleflight

import "sync"

// call 正在进行中或者已经结束的请求,使用 sync.WaitGroup 锁避免重入
type call struct {
	wg sync.WaitGroup
	val interface{}
	err error
}

// Group 是 singleflight 的主数据结构，管理不同key的请求 call
type Group struct {
	mu sync.Mutex
	m map[string]*call
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {			//延迟初始化，提高内存效率
		g.m = make(map[string]*call)
	}

	if c, ok := g.m[key]; ok {		//如果g.m中存在要查询的key
		g.mu.Unlock()				//那么就代表不需要修改g.m，解锁
		c.wg.Wait()					//Wait等待，直到可以获取c为止
		return c.val, c.err
	}

	c := new(call)
	c.wg.Add(1)				//发起请求前加锁
	g.m[key] = c
	g.mu.Unlock()					//修改完g.m之后就可以释放g.mu了

	c.val, c.err = fn()				//调用fn，发起请求
	c.wg.Done()						//请求结束，释放锁

	g.mu.Lock()
	delete(g.m, key)				//c的值获取完成后将key剔除，以方便之后可能出现的对key再次修改的请求
	g.mu.Unlock()

	return c.val, c.err
}


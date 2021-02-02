package selector

import (
	"sync"
	"time"
)

// 轮询算法
type roundRobinBalancer struct {
	pickers  *sync.Map     // 所有服务的服务名和其对应的服务器列表 保证并发安全性
	duration time.Duration // 更新间隔(多长时间更新一次)
}

type roundRobinPicker struct {
	length         int           // 服务列表的长度
	lastUpdateTime time.Time     // 上次访问时间
	duration       time.Duration // 更新间隔(多长时间更新一次) 继承自 roundRobinBalancer.duration
	lastIndex      int           // 上次访问下标
}

//从一个服务列表里面去获取一个服务节点
func (rp *roundRobinPicker) pick(nodes []*Node) *Node {
	if len(nodes) == 0 {
		return nil
	}

	// 超过时间后或者节点数量有变化时更新
	if time.Now().Sub(rp.lastUpdateTime) > rp.duration || len(nodes) != rp.length {
		rp.length = len(nodes)
		rp.lastUpdateTime = time.Now()
		rp.lastIndex = 0
	}

	// 两层含义：1、如果只有一个节点就返回第一个节点 2、如果到最后一个节点，下标归零
	if rp.lastIndex == len(nodes)-1 {
		rp.lastIndex = 0
		return nodes[0]
	}

	//节点加1
	rp.lastIndex += 1
	return nodes[rp.lastIndex]
}

func (r *roundRobinBalancer) Balance(serviceName string, nodes []*Node) *Node {

	var picker *roundRobinPicker

	// 加载节点信息
	if p, ok := r.pickers.Load(serviceName); !ok { //未找到就初始化
		picker = &roundRobinPicker{
			lastUpdateTime: time.Now(),
			duration:       r.duration,
			length:         len(nodes),
		}
	} else {
		picker = p.(*roundRobinPicker)
	}

	node := picker.pick(nodes)           //获取访问的服务节点
	r.pickers.Store(serviceName, picker) //存储
	return node
}

func newRoundRobinBalancer() *roundRobinBalancer {
	return &roundRobinBalancer{
		pickers:  new(sync.Map),   //初始化为并发安全的空 map
		duration: 3 * time.Minute, // 3秒
	}
}

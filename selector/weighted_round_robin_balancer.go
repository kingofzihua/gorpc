package selector

import (
	"sync"
	"time"
)

// 加权轮询算法
type weightedRoundRobinBalancer struct {
	pickers  *sync.Map     // 所有服务的服务名和其对应的服务器列表 保证并发安全性
	duration time.Duration // 更新间隔(多长时间更新一次)
}

func newWeightedRoundRobinBalancer() *weightedRoundRobinBalancer {
	return &weightedRoundRobinBalancer{
		pickers:  new(sync.Map),
		duration: 3 * time.Minute, // 3秒
	}
}

//服务节点权重
type weightedNode struct {
	node            *Node
	weight          int //节点权重
	effectiveWeight int //节点的有效权重 => 默认是节点权重
	currentWeight   int //节点的当前权重 => 默认是节点权重
}

type wRoundRobinPicker struct {
	nodes          []*weightedNode // 服务节点
	lastUpdateTime time.Time       // 最后更新时间
	duration       time.Duration   // 更新间隔
}

func (wr *wRoundRobinPicker) pick(nodes []*Node) *Node {
	if len(nodes) == 0 {
		return nil
	}

	// 当超过更新间隔或者节点发生变化的时候
	if time.Now().Sub(wr.lastUpdateTime) > wr.duration || len(nodes) != len(wr.nodes) {
		wr.nodes = getWeightedNode(nodes)
		wr.lastUpdateTime = time.Now()
	}

	totalWeight := 0 //总权重
	maxWeight := 0   //最大权重
	index := 0       //选中的节点下标
	for i, node := range wr.nodes {
		node.currentWeight += node.weight //每个节点，用它们当前的值加上自己的权重
		totalWeight += node.weight
		if node.currentWeight > maxWeight {
			maxWeight = node.currentWeight
			index = i
		}
	}

	//当前值最大的节点，把它的当前值减去所有节点的权重总和，作为它的新权重
	wr.nodes[index].currentWeight -= totalWeight

	return wr.nodes[index].node

}

func (w *weightedRoundRobinBalancer) Balance(serviceName string, nodes []*Node) *Node {
	var picker *wRoundRobinPicker

	if p, ok := w.pickers.Load(serviceName); !ok {
		picker = &wRoundRobinPicker{
			lastUpdateTime: time.Now(),
			duration:       w.duration,
			nodes:          getWeightedNode(nodes),
		}
		w.pickers.Store(serviceName, picker)
	} else {
		picker = p.(*wRoundRobinPicker)
	}

	node := picker.pick(nodes)
	w.pickers.Store(serviceName, picker)
	return node
}

// 获取加权节点
func getWeightedNode(nodes []*Node) []*weightedNode {

	var wgs []*weightedNode
	for _, node := range nodes {
		wgs = append(wgs, &weightedNode{
			node:            node,
			weight:          node.weight,
			currentWeight:   node.weight, // 默认是节点权重
			effectiveWeight: node.weight, // 默认是节点权重
		})
	}

	return wgs
}

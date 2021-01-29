package selector

// 服务节点的基本信息
type Node struct {
	Key    string
	Value  []byte
	weight int //权重
}

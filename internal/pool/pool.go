package pool

import (
	"k8s.io/apimachinery/pkg/labels"
)

type Node struct {
	labels *labels.Set
}

func (n *Node) GetLabels() *labels.Set {
	return n.labels
}

type pool struct {
	size               uint32
	allocationStrategy string
	nodes              map[string]*Node
}

type Pool interface {
	QueryByLabels(labels.Selector) []string
	QueryByName(string) []string
	Allocate(string, map[string]string) (string, bool)
	DeAllocate(string)
	GetAllocated() []string
}

func New(size uint32, allocStrategy string) Pool {
	return &pool{
		size:               size,
		allocationStrategy: allocStrategy,
		nodes:              make(map[string]*Node),
	}
}

func (p *pool) QueryByLabels(selector labels.Selector) []string {
	matches := make([]string, 0)
	for niname, n := range p.nodes {
		if selector.Matches(n.GetLabels()) {
			matches = append(matches, niname)
		}
	}
	return matches
}

func (p *pool) QueryByName(niname string) []string {
	matches := make([]string, 0)
	if _, ok := p.nodes[niname]; ok {
		matches = append(matches, niname)
	}
	return matches
}

func (p *pool) Allocate(key string, label map[string]string) (string, bool) {
	// TODO index based allocation
	switch p.allocationStrategy {
	default:
		// allocation strategy = first-available
		// key is used to match the pool key

		mergedlabel := labels.Merge(labels.Set(label), nil)

		p.nodes[key] = &Node{
			labels: &mergedlabel,
		}
		return key, true

	}
	//return "", false
}

func (p *pool) DeAllocate(key string) {
	delete(p.nodes, key)
}

func (p *pool) GetAllocated() []string {
	allocated := make([]string, 0)
	for key := range p.nodes {
		allocated = append(allocated, key)
	}
	return allocated
}

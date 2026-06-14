package router

type node struct {
	children map[byte]*node
	handler  interface{}
	prefix   string
}

type Htrie struct {
	root *node
}

func New() *Htrie {
	return &Htrie{root: &node{children: make(map[byte]*node)}}
}

func (t *Htrie) Insert(pattern string, handler interface{}) {
	n := t.root
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		if child, ok := n.children[ch]; ok {
			n = child
		} else {
			newNode := &node{children: make(map[byte]*node)}
			n.children[ch] = newNode
			n = newNode
		}
	}
	n.handler = handler
	n.prefix = pattern
}

func (t *Htrie) Search(path string) (interface{}, bool) {
	n := t.root
	var lastMatch *node
	var lastMatchLen int

	for i := 0; i < len(path); i++ {
		ch := path[i]
		child, ok := n.children[ch]
		if !ok {
			break
		}
		n = child
		if n.handler != nil {
			lastMatch = n
			lastMatchLen = i + 1
		}
	}

	if lastMatch != nil {
		if lastMatchLen == len(path) {
			return lastMatch.handler, true
		}
		if len(lastMatch.prefix) > 0 && lastMatch.prefix[len(lastMatch.prefix)-1] == '/' {
			return lastMatch.handler, true
		}
	}

	return nil, false
}

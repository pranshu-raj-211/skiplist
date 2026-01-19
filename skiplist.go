package skiplist

import (
	"errors"
	"math/rand"
)

type Node struct {
	key   int
	value int
	next  []*Node
}

type Skiplist struct {
	head     *Node
	level    int
	maxLevel int
	p        float64
}

// Creates a new skiplist for the given max level and promotion probability
func New(maxLevel int, p float64) *Skiplist {
	// maxLevel 0 means a normal linked list
	if maxLevel < 0 {
		maxLevel = 0
	}
	// default probability should be 1/2 - coin flip
	if p <= 0.0 || p >= 1.0 {
		p = 0.5
	}

	// keeping head's key as -1 does not matter as head's key is never compared
	// head node's key can be anything, we generally give it a negative infinity value for sentinel behavior
	head := &Node{
		key:  -1,
		next: make([]*Node, maxLevel+1),
	}

	return &Skiplist{
		head:     head,
		level:    0,
		maxLevel: maxLevel,
		p:        p,
	}
}

// Search for node with given key in the skiplist
func (s *Skiplist) Search(key int) (int, error) {
	current := s.head

	for i := s.level; i >= 0; i-- {
		for current.next[i] != nil && current.next[i].key < key {
			current = current.next[i]
		}
	}

	current = current.next[0]
	if current != nil && current.key == key {
		return current.value, nil
	}

	return 0, errors.New("key not found")
}

// Insert or update a key-value pair
func (s *Skiplist) Insert(key int, value int) {
	update := make([]*Node, s.maxLevel+1)
	current := s.head

	for i := s.level; i >= 0; i-- {
		for current.next[i] != nil && current.next[i].key < key {
			current = current.next[i]
		}
		update[i] = current
	}

	current = current.next[0]
	if current != nil && current.key == key {
		current.value = value
		return
	}

	nodeLevel := randomLevel(s.maxLevel, s.p)
	if nodeLevel > s.level {
		for i := s.level + 1; i <= nodeLevel; i++ {
			update[i] = s.head
		}
		s.level = nodeLevel
	}

	newNode := &Node{
		key:   key,
		value: value,
		next:  make([]*Node, nodeLevel+1),
	}

	for i := 0; i <= nodeLevel; i++ {
		newNode.next[i] = update[i].next[i]
		update[i].next[i] = newNode
	}
}

// Delete node with given key in the skiplist
func (s *Skiplist) Delete(key int) error {
	update := make([]*Node, s.maxLevel+1)
	current := s.head

	for i := s.level; i >= 0; i-- {
		for current.next[i] != nil && current.next[i].key < key {
			current = current.next[i]
		}
		update[i] = current
	}

	current = current.next[0]
	if current == nil || current.key != key {
		return errors.New("key not found")
	}

	for i := 0; i <= s.level; i++ {
		if update[i].next[i] != current {
			break
		}
		update[i].next[i] = current.next[i]
	}

	for s.level > 0 && s.head.next[s.level] == nil {
		s.level--
	}

	return nil
}

// Returns a random level based on a geometric probability distribution with prob p
// a fraction p of nodes from the current level will be promoted to next upper level
func randomLevel(maxLevel int, p float64) int {
	level := 0
	for level < maxLevel && rand.Float64() < p {
		level++
	}
	return level
}

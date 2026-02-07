package skiplist

import (
	"errors"
	"math"
	"math/rand"
	"sync"
)

type Node struct {
	key   int
	value interface{}
	next  []*Node
}

type Skiplist struct {
	head     *Node
	level    int
	maxLevel int
	p        float64
	probTable  []float64
	randsource *rand.Rand
}

var updatePool = sync.Pool{
	New: func() any {
		// Pre-allocate to maxLevel (e.g., 32)
		return make([]*Node, 32)
	},
}

// Creates a new skiplist for the given max level and promotion probability
func New(maxLevel int, p float64) *Skiplist {
	// maxLevel 0 means a normal linked list
	if maxLevel < 0 {
		maxLevel = 0
	}
	// default probability should be 1/2 - coin flip
	if p <= 0.0 || p >= 1.0 {
		p = 1 / math.E
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
		probTable: computeProbTable(maxLevel, p),
		randsource: rand.New(rand.NewSource(42)),
	}
}

// Search for node with given key in the skiplist
func (s *Skiplist) Search(key int) (interface{}, error) {
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
func (s *Skiplist) Insert(key int, value interface{}) {
	update := updatePool.Get().([]*Node)
	defer updatePool.Put(update)
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

	nodeLevel := s.randomLevel()
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
	update := updatePool.Get().([]*Node)
	defer updatePool.Put(update)
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
func (s *Skiplist) randomLevel() int {
	r:= s.randsource.Float64()
	level := 0
	for level < s.maxLevel && r < s.probTable[level] {
		level++
	}
	return level
}

// compute the probabilities at which we will promote in advance
func computeProbTable(maxLevel int, p float64) []float64 {
	probTable := make([]float64, maxLevel+1)
	currentProb := 1.0
	for i :=0; i<=maxLevel; i++{
		probTable[i]=currentProb
		currentProb*=p
	}
	return probTable
}
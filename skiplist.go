/*
Skiplist implementation according to the 1986 Pugh paper

Implement the search, insert and delete operations as defined in the given algorithm,
with 0 indexing used.
*/

package skiplist

import (
	"errors"
	"math/rand"
)

type Node struct {
	key   int
	value int
	level int
	next  []*Node
}

type Skiplist struct {
	maxLevel int
	level    int
	head     *Node
	p        float64
}

// TODO: review update logic, expecting off by one errors

// Creates a new skiplist for the given max level and promotion probability
func(s *Skiplist) New(maxLevel int, p float64) *Skiplist{
	// maxLevel 0 means a normal linked list
	if maxLevel<0{
		maxLevel=0
	}
	// default probability of level promotion is 1/2 (coin flip)
	if p<=0.0{
		p=0.5
	}

	return &Skiplist{maxLevel: maxLevel, level: 0, head: nil, p: p}
}

// Search for node with given key in the skiplist
func (s *Skiplist) Search(key int) (int, error) {
	current := s.head
	i := s.level
	for ; i >= 0; i-- {
		for current.next[i].key < key {
			current = current.next[i]
		}
	}
	current = current.next[i]
	if current.key == key {
		return current.value, nil
	}
	return 0, errors.New("could not find key in list")
}

// Insert a key value pair into an existing skiplist
func (s *Skiplist) Insert(key int, value int) error {
	current := s.head
	update := make([]*Node, s.maxLevel)

	i := s.level
	for ; i >= 0; i-- {
		for current.next[i].key < key {
			current = current.next[i]
		}
	}
	current = current.next[i]
	if current.key == key {
		current.value = value
	} else {
		level := randomLevel(s.maxLevel, s.p)
		if level > s.level {
			for i := s.level + 1; i < level; i++ {
				update[i] = s.head
			}
			s.level = level
		}
		current = &Node{level: level, key: key, value: value}
		for i := 1; i < level; i++ {
			current.next[i] = update[i].next[i]
			update[i].next[i] = current
		}
	}
	// ? do we need errors here
	return nil
}

// Delete node with given key in the skiplist
func (s *Skiplist) Delete(key int) error {
	current := s.head
	update := make([]*Node, s.maxLevel)

	i := s.level
	for ; i >= 0; i-- {
		for current.next[i].key < key {
			current = current.next[i]
		}
	}
	current = current.next[i]

	if current.key == key {
		// TODO: understand delete pointer splice logic
		for i := range s.level {
			if update[i].next[i] != current {
				break
			}
			update[i].next[i] = current.next[i]
		}
	} else {
		return errors.New("key not found")
	}
	// TODO: delete the current node
	for (s.level > 0) && (s.head.next[s.level]) == nil {
		s.level = s.level - 1
	}
	return nil
}

// TODO: the update slice needs to be updated each time we stop moving at a particular level - record
// the pointer to the point we stopped looking and moved to a lower level.


// Returns a random level based on a geometric probability distribution with prob p
// a fraction p of nodes from the current level will be promoted to next upper level
func randomLevel(maxLevel int, p float64) int {
	level := 0
	// set p to 1/2 in case value not set correctly
	if p == 0.0 {
		p = 0.5
	}
	for ; level < maxLevel; level++ {
		rand := rand.Float64()
		if rand > p {
			return level
		}
	}
	return level
}

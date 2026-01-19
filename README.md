## Skiplist

This package implements a skiplist as per the paper by Pugh - [link](https://15721.courses.cs.cmu.edu/spring2018/papers/08-oltpindexes1/pugh-skiplists-cacm1990.pdf)

I built this to understand concretely how skiplists worked, and also to implement sorted sets based on this, which I would use to replace the Redis usage in my [real time leaderboard project](https://github.com/pranshu-raj-211/leaderboard).

---

## What it is
A skiplist is a probabilistic data structure that's a sorted linked list with some added steps, to make searches faster.

Normally linked lists have search in the order of N, the number of elements in the linked list. Skip list contain additional pointers besides the normal linked list pointers, that allow them to "skip" nodes - go to a node that's much farther in the order, determined by a geometric probability distribution in this case.

This makes skiplist search time to be logarithmic in the average case, and while it is still linear in the worst case it's usually fine, especially when you compare the alternative (balanced binary trees) are much more complex to implement (the reason why Redis used skip lists to implement their sorted sets).

Inserts and Deletes are a bit more expensive than linked lists, since you have to update multiple pointers (splicing multiple pointers compared to a single one in a linked list) but when compared to the cost of the search that is needed for insert and delete it's a trivial difference anyways (maximum level is a small constant).

## How to use
This implementation uses a key value structure, where both of those are float32 values (since I want to support floating point values for my use case).

### Creation
A skiplist can be created by:

```go
s := skiplist.New(32, 0.5)
```

Where 32 is the maximum number of levels a skiplist can go to, and 0.5 is the probability that a node is promoted to the next level (intuition - a fraction p of nodes from a level l will be promoted to the level l+1).

This uses a geometric distribution as intended in the original paper, as having higher level nodes should be rarer. All nodes should have the level 0 pointer anyways (normal linked list pointer). Node levels are decided at node creation time (insert).

### Search
Search for the value corresponding to a key k in skiplist s.

```go
s.Search(k)
```

Returns the float32 value and a nil if key exists in s, otherwise returns 0 and an error with the message "key not found".

### Insert
Insert a key value pair into an existing skip list s by:

```go
s.Insert(a, b)
```

where a and b are float32 values.

### Update
Update the value of an existing key, call insert with the same key, just changes the value (pointers unchanged).

```go
s.Insert(a, b)
```

where a and b are float32 values and a is an existing key in the skip list s.

Note - if a key does not exist a node corresponding to it will be added, there cannot be duplicate keys in the skip list.

### Delete
Delete a node with a key k from the skip list s if exists.

```go
s.Delete(k)
```

Returns a nil if the key existed in the skip list, otherwise an error with the message "key not found".

## Perf
Still refining tests, making them more comprehensive, but initial numbers look pretty good.

### Search


### Insertion

#### Sequential inserts (insertion key part of a known sequence)
Inserting a single element into a skip list (with sequential keys), with 1 million nodes allocated already gets these results (note - was using int when these benchmarks were created).

`199.8 ns/op	      72 B/op	       3 allocs/op`

as averaged over 7140824 runs

The performance is 200±5ns for skip lists with 1000, 100k and 1 million nodes. Interestingly it gets to 231ns for 10 nodes. I suspect it has to do with the fact that there's too many test runs and something wrong with my testing logic (newbie at bench tests).

#### Random inserts (insertion keys are random values)
This takes a lot more time, another order of magnitude over the sequential inserts, so its probably the random checks taking up time.

Inserting a random key value pair into a skip list with 1 million elements pre allocated takes `1715 ns/op	      69 B/op	       2 allocs/op` averaged over 1000000 runs. The lower allocation sizes fair no better, with 1259ns, 1164ns, 1223ns for 100k, 1000, 10 elements respectively.

We should also consider that since the elements being inserted are random, there may be collisions (though rare). Also need to see distribution of insertion keys to make it fairly distributed over a large space.

### Search
#### Search random keys
Search takes 1529ns, averaged over 710508 runs for a skiplist with 1 million nodes in it, with keys for search being generated randomly.

### Delete
Delete operations are still being benchmarked, with a somewhat larger than expected time of delete (doing random deletes), nevertheless it currently stands at 1135ns per delete.

---

There's a lot of issues with the benchmarking code obviously, so fixing that and releasing a good benchmark will be top priority for this project. A sample implementation of the sorted set is ready, though I have not referenced the Redis implementation at all and have built it according to my own ideas, so I need to read more on that and then look at releasing that package as well.
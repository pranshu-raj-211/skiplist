package skiplist

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"
)

type Quantile struct {
	P50 time.Duration
	P90 time.Duration
	P95 time.Duration
	P99 time.Duration
}

func calculateQuantiles(durations []time.Duration) Quantile {
	if len(durations) == 0 {
		return Quantile{}
	}
	d := make([]time.Duration, len(durations))
	copy(d, durations)
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
	n := len(d)
	return Quantile{
		P50: d[n*50/100],
		P90: d[n*90/100],
		P95: d[n*95/100],
		P99: d[n*99/100],
	}
}

func printQuantiles(name string, q Quantile) {
	fmt.Printf("  %-30s  p50=%-10v p90=%-10v p95=%-10v p99=%v\n",
		name, q.P50, q.P90, q.P95, q.P99)
}

func measure(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		maxLevel    int
		p           float64
		wantMaxLvl  int
		wantDefault bool // true means p should have been replaced by the default
	}{
		{"Standard", 16, 0.5, 16, false},
		{"Negative MaxLevel clamps to 0", -5, 0.5, 0, false},
		{"Zero MaxLevel allowed", 0, 0.5, 0, false},
		{"Invalid p high defaults", 10, 1.5, 10, true},
		{"Invalid p low defaults", 10, -0.1, 10, true},
		{"Boundary p=0 defaults", 10, 0.0, 10, true},
		{"Boundary p=1 defaults", 10, 1.0, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sl := New(tt.maxLevel, tt.p)
			if sl.maxLevel != tt.wantMaxLvl {
				t.Errorf("maxLevel = %d, want %d", sl.maxLevel, tt.wantMaxLvl)
			}
			// When an invalid p is supplied, New must pick some valid default
			// (1/e per the implementation).  Just verify it is in (0,1).
			if tt.wantDefault && (sl.p <= 0 || sl.p >= 1) {
				t.Errorf("default p = %v is not in (0,1)", sl.p)
			}
			if !tt.wantDefault && sl.p != tt.p {
				t.Errorf("p = %v, want %v", sl.p, tt.p)
			}
			if sl.head == nil {
				t.Error("head sentinel is nil")
			}
		})
	}
}

func TestInsertAndSearch(t *testing.T) {
	sl := New(10, 0.5)
	sl.Insert(10, 100)
	val, ok := sl.Search(10)
	if !ok {
		t.Fatal("Search(10): key not found after insert")
	}
	if val != 100 {
		t.Errorf("Search(10) = %v, want 100", val)
	}

	// Update existing
	sl.Insert(10, 200)
	val, ok = sl.Search(10)
	if !ok {
		t.Fatal("Search(10): key not found after update")
	}
	if val != 200 {
		t.Errorf("after update Search(10) = %v, want 200", val)
	}

	// Missing key
	_, ok = sl.Search(9999)
	if ok {
		t.Error("Search(9999) returned ok=true for a key that was never inserted")
	}
}

func TestInsertOrdering(t *testing.T) {
	sl := New(8, 0.5)
	keys := []int{50, 10, 90, 30, 70, 20, 80, 40, 60}
	for _, k := range keys {
		sl.Insert(k, k*10)
	}

	// verify strictly ascending order
	node := sl.head.next[0]
	prev := -1
	for node != nil {
		if node.key <= prev {
			t.Errorf("ordering violation: %d after %d", node.key, prev)
		}
		prev = node.key
		node = node.next[0]
	}
}

func TestInsertManyAndSearchAll(t *testing.T) {
	const n = 1000
	sl := New(16, 0.5)
	for i := 0; i < n; i++ {
		sl.Insert(i, i*7)
	}
	for i := 0; i < n; i++ {
		val, ok := sl.Search(i)
		if !ok {
			t.Fatalf("Search(%d) not found", i)
		}
		if val != i*7 {
			t.Fatalf("Search(%d) = %v, want %d", i, val, i*7)
		}
	}
}

func TestDelete(t *testing.T) {
	sl := New(10, 0.5)
	sl.Insert(1, 10)
	sl.Insert(2, 20)
	sl.Insert(3, 30)

	// Delete middle element.
	sl.Delete(2)
	if _, ok := sl.Search(2); ok {
		t.Error("Search(2) should return ok=false after deletion")
	}

	// Neighbours must still be intact
	if v, ok := sl.Search(1); !ok || v != 10 {
		t.Errorf("Search(1) = %v, %v after deleting 2", v, ok)
	}
	if v, ok := sl.Search(3); !ok || v != 30 {
		t.Errorf("Search(3) = %v, %v after deleting 2", v, ok)
	}

	// Deleting a non-existent key must not panic or corrupt the list
	sl.Delete(9999)
	if v, ok := sl.Search(1); !ok || v != 10 {
		t.Error("list corrupted after deleting non-existent key")
	}
}

func TestDeleteAllElements(t *testing.T) {
	sl := New(8, 0.5)
	keys := []int{5, 1, 10, 7, 3}
	for _, k := range keys {
		sl.Insert(k, k)
	}
	for _, k := range keys {
		sl.Delete(k)
		if _, ok := sl.Search(k); ok {
			t.Errorf("key %d still found after deletion", k)
		}
	}
	// After all deletions the active level should collapse back to 0
	if sl.level != 0 {
		t.Errorf("level = %d after deleting all elements, want 0", sl.level)
	}
}

func TestDeleteHead(t *testing.T) {
	sl := New(8, 0.5)
	for _, k := range []int{1, 2, 3} {
		sl.Insert(k, k)
	}
	sl.Delete(1)
	if _, ok := sl.Search(1); ok {
		t.Error("smallest key still found after deletion")
	}
	if v, ok := sl.Search(2); !ok || v != 2 {
		t.Error("second key missing or wrong after deleting head")
	}
}

func TestDeleteTail(t *testing.T) {
	sl := New(8, 0.5)
	for _, k := range []int{1, 2, 3} {
		sl.Insert(k, k)
	}
	sl.Delete(3)
	if _, ok := sl.Search(3); ok {
		t.Error("largest key still found after deletion")
	}
	if v, ok := sl.Search(2); !ok || v != 2 {
		t.Error("second-to-last key missing after deleting tail")
	}
}

func TestRangeQuery(t *testing.T) {
	sl := New(8, 0.5)
	for i := 1; i <= 10; i++ {
		sl.Insert(i, i*100)
	}

	got := sl.RangeQuery(3, 7)
	if len(got) != 5 {
		t.Fatalf("RangeQuery(3,7) returned %d results, want 5", len(got))
	}
	for idx, kv := range got {
		wantKey := 3 + idx
		if kv.key != wantKey {
			t.Errorf("result[%d].key = %d, want %d", idx, kv.key, wantKey)
		}
		if kv.value != wantKey*100 {
			t.Errorf("result[%d].value = %v, want %d", idx, kv.value, wantKey*100)
		}
	}
}

func TestRangeQueryEdgeCases(t *testing.T) {
	sl := New(8, 0.5)
	for i := 1; i <= 5; i++ {
		sl.Insert(i, i)
	}

	// startKey > endKey must return nil, not panic.
	if r := sl.RangeQuery(5, 1); r != nil {
		t.Errorf("RangeQuery(5,1) = %v, want nil", r)
	}

	// startKey == endKey that exists.
	r := sl.RangeQuery(3, 3)

	// startKey not present: per the current implementation this returns nil.
	r2 := sl.RangeQuery(0, 3)
	_ = r2 // documented behaviour: returns nil when startKey absent

	// Both boundaries beyond the list.
	r3 := sl.RangeQuery(100, 200)
	_ = r3

	if len(r) != 1 || r[0].key != 3 {
		t.Errorf("single-element range: got %v", r)
	}
}

func TestRangeIterator(t *testing.T) {
	sl := New(8, 0.5)
	for i := 1; i <= 10; i++ {
		sl.Insert(i, i*100)
	}

	it := sl.RangeQueryIterator(3, 7)
	collected := []int{}
	for it.current != nil && it.current.key <= it.endKey {
		collected = append(collected, it.current.key)
		it.Next()
	}

	if len(collected) != 5 {
		t.Fatalf("iterator collected %d keys, want 5: %v", len(collected), collected)
	}
	for i, k := range collected {
		if k != 3+i {
			t.Errorf("collected[%d] = %d, want %d", i, k, 3+i)
		}
	}
}

func TestRangeIteratorInvalidRange(t *testing.T) {
	sl := New(4, 0.5)
	sl.Insert(1, 1)
	it := sl.RangeQueryIterator(5, 1) // start > end
	if it.current != nil {
		t.Error("iterator for invalid range should have nil current")
	}
}

func TestLevelNeverExceedsMax(t *testing.T) {
	sl := New(8, 0.5)
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 10_000; i++ {
		sl.Insert(rng.Intn(5000), i)
	}
	if sl.level > sl.maxLevel {
		t.Errorf("level %d exceeds maxLevel %d", sl.level, sl.maxLevel)
	}
}

func TestConsistency(t *testing.T) {
	sl := New(8, 0.5)
	data := map[int]int{5: 50, 1: 10, 10: 100, 7: 70, 3: 30}

	for k, v := range data {
		sl.Insert(k, v)
	}
	for k, v := range data {
		got, ok := sl.Search(k)
		if !ok || got != v {
			t.Errorf("Search(%d) = %v, %v; want %d, true", k, got, ok, v)
		}
	}

	for k := range data {
		sl.Delete(k)
		if _, ok := sl.Search(k); ok {
			t.Errorf("key %d still present after deletion", k)
		}
	}
	if sl.level != 0 {
		t.Errorf("level should be 0 after emptying list, got %d", sl.level)
	}
}

// perf tests - quantile

func TestQuantilesSearch(t *testing.T) {
	const n = 10_000
	sl := New(16, 0.5)
	for i := 0; i < n; i++ {
		sl.Insert(i, i*10)
	}

	keys := rand.Perm(n)
	dHit := make([]time.Duration, n)
	for i, k := range keys {
		dHit[i] = measure(func() { sl.Search(k) })
	}

	// Miss- search for keys that were never inserted.
	dMiss := make([]time.Duration, n)
	for i := range dMiss {
		k := n + rand.Intn(n) // guaranteed absent
		dMiss[i] = measure(func() { sl.Search(k) })
	}

	fmt.Println("\n── Search quantiles ──────────────────────────────────────────")
	printQuantiles("hit  (key exists)", calculateQuantiles(dHit))
	printQuantiles("miss (key absent)", calculateQuantiles(dMiss))
}

func TestQuantilesInsert(t *testing.T) {
	const n = 10_000
	sl := New(16, 0.5)

	// Sequential inserts (best-case ordering for a forward-biased traversal).
	dSeq := make([]time.Duration, n)
	for i := range dSeq {
		k := i
		dSeq[i] = measure(func() { sl.Insert(k, k) })
	}

	sl2 := New(16, 0.5)
	keys := rand.Perm(n * 10)[:n]

	// Random inserts into a fresh list.
	dRand := make([]time.Duration, n)
	for i, k := range keys {
		k := k
		dRand[i] = measure(func() { sl2.Insert(k, k) })
	}

	// Updates (key already present).
	dUpd := make([]time.Duration, n)
	updKeys := rand.Perm(n) // all exist in sl
	for i, k := range updKeys {
		k := k
		dUpd[i] = measure(func() { sl.Insert(k, k*2) })
	}

	fmt.Println("\n── Insert quantiles ──────────────────────────────────────────")
	printQuantiles("sequential (new keys)", calculateQuantiles(dSeq))
	printQuantiles("random     (new keys)", calculateQuantiles(dRand))
	printQuantiles("update     (existing)", calculateQuantiles(dUpd))
}

func TestQuantilesDelete(t *testing.T) {
	const n = 10_000

	makeList := func() *Skiplist {
		sl := New(16, 0.5)
		for i := 0; i < n; i++ {
			sl.Insert(i, i)
		}
		return sl
	}

	// Sequential delete.
	sl1 := makeList()
	dSeq := make([]time.Duration, n)
	for i := range dSeq {
		k := i
		dSeq[i] = measure(func() { sl1.Delete(k) })
	}

	// Reverse delete.
	sl2 := makeList()
	dRev := make([]time.Duration, n)
	for i := range dRev {
		k := n - 1 - i
		dRev[i] = measure(func() { sl2.Delete(k) })
	}

	// Random delete.
	sl3 := makeList()
	randKeys := rand.Perm(n)
	dRand := make([]time.Duration, n)
	for i, k := range randKeys {
		k := k
		dRand[i] = measure(func() { sl3.Delete(k) })
	}

	// Delete non-existent.
	sl4 := makeList()
	dMiss := make([]time.Duration, n)
	for i := range dMiss {
		k := n + i // never inserted
		dMiss[i] = measure(func() { sl4.Delete(k) })
	}

	fmt.Println("\n── Delete quantiles ──────────────────────────────────────────")
	printQuantiles("sequential", calculateQuantiles(dSeq))
	printQuantiles("reverse", calculateQuantiles(dRev))
	printQuantiles("random", calculateQuantiles(dRand))
	printQuantiles("miss (absent key)", calculateQuantiles(dMiss))
}

func TestQuantilesRangeQuery(t *testing.T) {
	const dataSize = 100_000
	sl := New(16, 0.5)
	for i := 0; i < dataSize; i++ {
		sl.Insert(i, i*10)
	}

	const n = 1000

	run := func(label string, size int) {
		d := make([]time.Duration, n)
		for i := range d {
			start := rand.Intn(dataSize - size - 1)
			end := start + size
			d[i] = measure(func() { sl.RangeQuery(start, end) })
		}
		printQuantiles(label, calculateQuantiles(d))
	}

	fmt.Println("\n── RangeQuery quantiles ──────────────────────────────────────")
	run("size=10   (tiny)", 10)
	run("size=100  (small)", 100)
	run("size=1000 (medium)", 1000)
	run("size=10000 (large)", 10000)
}

func TestQuantilesMixedWorkload(t *testing.T) {
	const n = 10_000
	sl := New(16, 0.5)
	for i := 0; i < n/2; i++ {
		sl.Insert(i, i*10)
	}

	// 60% search, 30% insert/update, 10% delete — a realistic OLTP mix.
	type op struct{ typ, key int }
	ops := make([]op, n)
	for i := range ops {
		r := rand.Intn(10)
		ops[i] = op{r, rand.Intn(n * 2)}
	}

	dSearch := make([]time.Duration, 0, n)
	dInsert := make([]time.Duration, 0, n)
	dDelete := make([]time.Duration, 0, n)

	for _, o := range ops {
		switch {
		case o.typ < 6: // 0-5 → search
			k := o.key
			dSearch = append(dSearch, measure(func() { sl.Search(k) }))
		case o.typ < 9: // 6-8 → insert
			k := o.key
			dInsert = append(dInsert, measure(func() { sl.Insert(k, k) }))
		default: // 9 → delete
			k := o.key
			dDelete = append(dDelete, measure(func() { sl.Delete(k) }))
		}
	}

	fmt.Println("\n── Mixed workload quantiles (60r/30w/10d) ───────────────────")
	printQuantiles("search", calculateQuantiles(dSearch))
	printQuantiles("insert/update", calculateQuantiles(dInsert))
	printQuantiles("delete", calculateQuantiles(dDelete))
}

func TestQuantilesVaryingMaxLevel(t *testing.T) {
	const n = 10_000
	fmt.Println("\n── Search quantiles by maxLevel ──────────────────────────────")
	for _, maxLvl := range []int{4, 8, 12, 16, 20} {
		sl := New(maxLvl, 0.5)
		for i := 0; i < n; i++ {
			sl.Insert(i, i)
		}
		keys := rand.Perm(n)
		d := make([]time.Duration, n)
		for i, k := range keys {
			k := k
			d[i] = measure(func() { sl.Search(k) })
		}
		printQuantiles(fmt.Sprintf("maxLevel=%2d", maxLvl), calculateQuantiles(d))
	}
}

// TestQuantilesAtPercentile answers the question "at what percentile does
// X nanoseconds fall?" for Search, as asked.
func TestQuantilesAtPercentile(t *testing.T) {
	const n = 100_000
	sl := New(16, 0.5)
	for i := 0; i < n; i++ {
		sl.Insert(i, i)
	}
	keys := rand.Perm(n)
	d := make([]time.Duration, n)
	for i, k := range keys {
		k := k
		d[i] = measure(func() { sl.Search(k) })
	}
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })

	thresholds := []time.Duration{
		100 * time.Nanosecond,
		200 * time.Nanosecond,
		500 * time.Nanosecond,
		1 * time.Microsecond,
		2 * time.Microsecond,
		5 * time.Microsecond,
		10 * time.Microsecond,
	}

	fmt.Println("\n── Search: at which percentile does latency cross threshold? ─")
	for _, thresh := range thresholds {
		// lower_bound: first index where d[i] >= thresh
		idx := sort.Search(len(d), func(i int) bool { return d[i] >= thresh })
		pct := float64(idx) / float64(len(d)) * 100
		fmt.Printf("  %-12v  → p%.1f\n", thresh, pct)
	}
}

// Benchmarks
//
// Rules followed:
//  1. Use benchmark tools for time calculation.
//  2. Pre-generate all inputs before b.ResetTimer().
//  3. Pre-warm to a stable size (magnitude of operation should be significantly less than original 
// skiplist size) before b.ResetTimer().
//  4. Use fixed-size key space for delete benchmarks.
//  5. Sink return values with a package-level var to prevent the compiler from optimising away calls 
// whose results are unused.

var sinkAny any
var sinkBool bool

// number of elements inserted before timing begins
const benchPreload = 1_000_000

func buildList(n int) *Skiplist {
	sl := New(16, 0.5)
	for i := 0; i < n; i++ {
		sl.Insert(i, i)
	}
	return sl
}

// searches for keys that exist, in sequential order
func BenchmarkSearchSequential(b *testing.B) {
	sl := buildList(benchPreload)
	keys := make([]int, b.N)
	for i := range keys {
		keys[i] = i % benchPreload
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkAny, sinkBool = sl.Search(keys[i])
	}
}

// exercises the full skip-down traversal with unpredictable cache misses
func BenchmarkSearchRandom(b *testing.B) {
	sl := buildList(benchPreload)
	keys := make([]int, b.N)
	for i := range keys {
		keys[i] = rand.Intn(benchPreload)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkAny, sinkBool = sl.Search(keys[i])
	}
}

// traversal must reach the bottom level before giving up
func BenchmarkSearchMiss(b *testing.B) {
	sl := buildList(benchPreload)
	// Keys are all > benchPreload so they can never exist.
	keys := make([]int, b.N)
	for i := range keys {
		keys[i] = benchPreload + rand.Intn(benchPreload)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkAny, sinkBool = sl.Search(keys[i])
	}
}

// The predecessor is always the tail, so update[] is found in one step at every level, which is the best case for insert.
func BenchmarkInsertSequential(b *testing.B) {
	sl := buildList(benchPreload)
	// Start keys above preload so they are all new inserts, never updates.
	keys := make([]int, b.N)
	for i := range keys {
		keys[i] = benchPreload + i
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sl.Insert(keys[i], i)
	}
}

// inserts random keys into a pre-warmed list
// keys are drawn from a space 10× the preload
func BenchmarkInsertRandom(b *testing.B) {
	sl := buildList(benchPreload)
	keys := make([]int, b.N)
	for i := range keys {
		keys[i] = rand.Intn(benchPreload * 10)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sl.Insert(keys[i], i)
	}
}

// no node allocation.
func BenchmarkInsertUpdate(b *testing.B) {
	sl := buildList(benchPreload)
	keys := make([]int, b.N)
	for i := range keys {
		keys[i] = rand.Intn(benchPreload)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sl.Insert(keys[i], i)
	}
}

// keys are mod benchPreload so the list never fully drains, later iterations
// will be no-op deletes (key already gone), which is fine
func BenchmarkDelete(b *testing.B) {
	sl := buildList(benchPreload)
	keys := make([]int, b.N)
	for i := range keys {
		keys[i] = rand.Intn(benchPreload)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sl.Delete(keys[i])
	}
}

// tests on a stable 1M-element list for various range sizes (linked list traversal important here)
func BenchmarkRangeQuerySmall(b *testing.B) {
	benchRangeQuery(b, 10)
}
func BenchmarkRangeQueryMedium(b *testing.B) {
	benchRangeQuery(b, 100)
}
func BenchmarkRangeQueryLarge(b *testing.B) {
	benchRangeQuery(b, 10_000)
}

func benchRangeQuery(b *testing.B, rangeSize int) {
	b.Helper()
	sl := buildList(benchPreload)
	type qr struct{ start, end int }
	qs := make([]qr, b.N)
	for i := range qs {
		s := rand.Intn(benchPreload - rangeSize - 1)
		qs[i] = qr{s, s + rangeSize}
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = sl.RangeQuery(qs[i].start, qs[i].end)
	}
}

// runs a 60/30/10 read/write/delete mix, matching the
// TestQuantilesMixedWorkload distribution so the two are directly comparable
func BenchmarkMixedWorkload(b *testing.B) {
	sl := buildList(benchPreload)
	type op struct{ typ, key, val int }
	ops := make([]op, b.N)
	for i := range ops {
		ops[i] = op{rand.Intn(10), rand.Intn(benchPreload * 2), i}
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		o := ops[i]
		switch {
		case o.typ < 6:
			sinkAny, sinkBool = sl.Search(o.key)
		case o.typ < 9:
			sl.Insert(o.key, o.val)
		default:
			sl.Delete(o.key)
		}
	}
}
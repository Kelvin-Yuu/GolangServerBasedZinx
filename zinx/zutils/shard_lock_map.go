package zutils

import (
	"encoding/json"
	"sync"
)

var ShardCount = 32

const (
	Prime   = 16777619
	HashVal = 2166136261
)

// ShardLockMaps：一种字符串类型的线程安全映射
// 为了避免锁定瓶颈，该Map被划分为多个（ShardCount个）Map碎片。
type ShardLockMaps struct {
	shards       []*SingleShardMap
	shardKeyFunc func(key string) uint32
}

// ShardLockMaps：一种字符串类型的线程安全映射
type SingleShardMap struct {
	items map[string]interface{}
	sync.RWMutex
}

// 创建ShardLockMaps
func createShardLockMaps(shardKeyFunc func(key string) uint32) ShardLockMaps {
	slm := ShardLockMaps{
		shards:       make([]*SingleShardMap, ShardCount),
		shardKeyFunc: shardKeyFunc,
	}

	for i := 0; i < ShardCount; i++ {
		slm.shards[i] = &SingleShardMap{items: make(map[string]interface{})}
	}

	return slm
}

// 创建ShardLockMaps
func NewShardLockMaps() ShardLockMaps {
	return createShardLockMaps(fnv32)
}

// 创建ShardLockMaps
func NewWithCustomShardKeyFunc(shardKeyFunc func(key string) uint32) ShardLockMaps {
	return createShardLockMaps(shardKeyFunc)
}

// FNV哈希算法是一种高离散性的哈希算法，特别适用于哈希非常相似的字符串，例如：URL，IP，主机名，文件名等。
// 该算法实现简单，特别适合互联网行业。
// 但该算法也有几个缺点
// 1. 不适用于加密，因为其执行效率高，容易攻击；
// 2. 由于hash结果是按位异或和乘积的，如果任何一步出现0，则结果可能会造成冲突；
func fnv32(key string) uint32 {
	hashVal := uint32(HashVal)
	prime := uint32(Prime)
	KeyLength := len(key)
	for i := 0; i < KeyLength; i++ {
		hashVal *= prime
		hashVal ^= uint32(key[i])
	}
	return hashVal
}

// 在给定Key下返回shard
func (slm ShardLockMaps) GetShard(key string) *SingleShardMap {
	return slm.shards[fnv32(key)%uint32(ShardCount)]
}

// 返回映射中的元素数
func (slm ShardLockMaps) Count() int {
	count := 0
	for i := 0; i < ShardCount; i++ {
		shard := slm.shards[i]
		shard.RLock()
		count += len(shard.items)
		shard.RUnlock()
	}
	return count
}

// 从给指定Key下的映射中检索元素
func (slm ShardLockMaps) Get(key string) (interface{}, bool) {
	shard := slm.GetShard(key)
	shard.RLock()
	val, ok := shard.items[key]
	shard.RUnlock()
	return val, ok
}

// 给指定的Key设置值
func (slm ShardLockMaps) Set(key string, value interface{}) {
	shard := slm.GetShard(key)
	shard.Lock()
	shard.items[key] = value
	shard.Unlock()
}

// 如果没有与指定Key关联的值，则设置该Key的值
func (slm ShardLockMaps) SetNX(key string, value interface{}) bool {
	shard := slm.GetShard(key)
	shard.Lock()
	_, ok := shard.items[key]
	if !ok {
		shard.items[key] = value
	}
	shard.Unlock()
	return !ok
}

// 设置多对Key-Value
func (slm ShardLockMaps) MSet(data map[string]interface{}) {
	for key, value := range data {
		shard := slm.GetShard(key)
		shard.Lock()
		shard.items[key] = value
		shard.Unlock()
	}
}

// 查找是否存在指定Key
func (slm ShardLockMaps) Has(key string) bool {
	shard := slm.GetShard(key)
	shard.RLock()
	_, ok := shard.items[key]
	shard.RUnlock()
	return ok
}

// 删除指定Key
func (slm ShardLockMaps) Remove(key string) {
	shard := slm.GetShard(key)
	shard.Lock()
	delete(shard.items, key)
	shard.Unlock()
}

// RemoveCb是在映射中执行的回调。RemoveCb()调用，同时持有Lock。
// 如果返回true，则该元素将从映射中删除。
type RemoveCb func(key string, v interface{}, exists bool) bool

// RemoveCb锁定包含Key的Shard，检索其当前值，并使用这些参数调用回调。
// 如果回调返回true并且元素存在，它将从映射中删除该元素。
// 返回回调返回的值（即使元素不在映射中）。
func (slm ShardLockMaps) RemoveCb(key string, cb RemoveCb) bool {
	shard := slm.GetShard(key)
	shard.Lock()
	v, ok := shard.items[key]
	remove := cb(key, v, ok)
	if remove && ok {
		delete(shard.items, key)
	}
	shard.Unlock()
	return remove
}

// 从映射中删除元素并返回
func (slm ShardLockMaps) Pop(key string) (v interface{}, exists bool) {
	shard := slm.GetShard(key)
	shard.Lock()
	v, exists = shard.items[key]
	delete(shard.items, key)
	shard.Unlock()
	return v, exists
}

// 删除所有元素
func (slm ShardLockMaps) Clear() {
	for item := range slm.IterBuffered() {
		slm.Remove(item.Key)
	}
}

// 检查Map是否为空
func (slm ShardLockMaps) IsEmpty() bool {
	return slm.Count() == 0
}

// 用于IterBuffered函数，在通道上将两个变量包装在一起。
type Tuple struct {
	Key string
	Val interface{}
}

// 返回一个通道数组，该数组包含每个碎片中的元素，这可能会获取“slm”的快照
// 一旦确定了每个缓冲通道的大小，在使用goroutines填充所有通道之前，它就会返回
func snapshot(slm ShardLockMaps) (chanList []chan Tuple) {
	chanList = make([]chan Tuple, ShardCount)
	wg := sync.WaitGroup{}
	wg.Add(ShardCount)
	for index, shard := range slm.shards {
		go func(index int, shard *SingleShardMap) {
			shard.RLock()
			chanList[index] = make(chan Tuple, len(shard.items))
			wg.Done()
			for key, val := range shard.items {
				chanList[index] <- Tuple{key, val}
			}
			shard.RUnlock()
			close(chanList[index])
		}(index, shard)
	}
	wg.Wait()
	return chanList
}

// 将元素从channel`chanList`读入channel`out`
func fanIn(chanList []chan Tuple, out chan Tuple) {
	wg := sync.WaitGroup{}
	wg.Add(len(chanList))
	for _, ch := range chanList {
		go func(ch chan Tuple) {
			for t := range ch {
				out <- t
			}
			wg.Done()
		}(ch)
	}
	wg.Wait()
	close(out)
}

// 返回一个缓冲迭代器，该迭代器可以在for循环中使用
func (slm ShardLockMaps) IterBuffered() <-chan Tuple {
	chanList := snapshot(slm)
	total := 0
	for _, c := range chanList {
		total += cap(c)
	}
	ch := make(chan Tuple, total)
	go fanIn(chanList, ch)
	return ch
}

// 返回所有item
func (slm ShardLockMaps) Items() map[string]interface{} {
	tmp := make(map[string]interface{})
	for item := range slm.IterBuffered() {
		tmp[item.Key] = item.Val
	}
	return tmp
}

// 返回所有Keys
func (slm ShardLockMaps) Keys() []string {
	count := slm.Count()
	ch := make(chan string, count)
	go func() {
		wg := sync.WaitGroup{}
		wg.Add(ShardCount)
		for _, shard := range slm.shards {
			go func(shard *SingleShardMap) {
				shard.RLock()
				for key := range shard.items {
					ch <- key
				}
				shard.RUnlock()
				wg.Done()
			}(shard)
		}
		wg.Wait()
		close(ch)
	}()

	keys := make([]string, 0, count)
	for k := range ch {
		keys = append(keys, k)
	}
	return keys
}

// 迭代器回调，为映射中找到的每个键和值而调用。
// RLock用于给定碎片的所有调用，因此回调shard的一致视图，但不跨碎片。
type IterCb func(key string, v interface{})

// 基于回调的迭代器，读取映射中所有元素的最简单的方法。
func (slm ShardLockMaps) IterCb(fn IterCb) {
	for idx := range slm.shards {
		shard := (slm.shards)[idx]
		shard.RLock()
		for key, value := range shard.items {
			fn(key, value)
		}
		shard.RUnlock()
	}
}

func (slm ShardLockMaps) MarshalJSON() ([]byte, error) {
	tmp := make(map[string]interface{})

	for item := range slm.IterBuffered() {
		tmp[item.Key] = item.Val
	}
	return json.Marshal(tmp)
}

func (slm ShardLockMaps) UnmarshalJSON(b []byte) (err error) {
	tmp := make(map[string]interface{})

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	for key, val := range tmp {
		slm.Set(key, val)
	}
	return nil
}

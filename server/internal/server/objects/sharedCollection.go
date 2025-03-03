package objects

import "sync"

// A generic, thread-safe map of objects with auto-incrementing IDs.
type SharedCollection[T any] struct {
	objectsMap map[uint64]T
	nextId     uint64
	mapMux     sync.Mutex
}

func NewSharedCollection[T any](capacity ...int) *SharedCollection[T] {
	var newObjMap map[uint64]T

	if len(capacity) > 0 {
		newObjMap = make(map[uint64]T, capacity[0])
	} else {
		newObjMap = make(map[uint64]T)
	}

	return &SharedCollection[T]{
		objectsMap: newObjMap,
		nextId:     1,
	}
}

func (c *SharedCollection[T]) Add(obj T, id ...uint64) uint64 {
	c.mapMux.Lock()
	defer c.mapMux.Unlock()

	thisId := c.nextId
	if len(id) > 0 {
		thisId = id[0]
	}
	c.objectsMap[thisId] = obj
	c.nextId++
	return thisId
}
func (c *SharedCollection[T]) Get(id uint64) (T, bool) {
	c.mapMux.Lock()
	defer c.mapMux.Unlock()

	obj, exists := c.objectsMap[id]
	return obj, exists
}

func (c *SharedCollection[T]) Remove(id uint64) {
	c.mapMux.Lock()
	defer c.mapMux.Unlock()

	delete(c.objectsMap, id)
}
func (s *SharedCollection[T]) ForEach(callback func(id uint64, obj T)) {
	s.mapMux.Lock()
	localCopy := make(map[uint64]T, len(s.objectsMap))
	for id, obj := range s.objectsMap {
		localCopy[id] = obj
	}
	s.mapMux.Unlock()

	for id, obj := range localCopy {
		callback(id, obj)
	}
}
func (c *SharedCollection[T]) GetAll() []T {
	c.mapMux.Lock()
	defer c.mapMux.Unlock()

	allObjects := make([]T, 0, len(c.objectsMap))
	for _, obj := range c.objectsMap {
		allObjects = append(allObjects, obj)
	}
	return allObjects
}

func (c *SharedCollection[T]) Len() int {
	c.mapMux.Lock()
	defer c.mapMux.Unlock()

	return len(c.objectsMap)
}

func (c *SharedCollection[T]) Clear() {
	c.mapMux.Lock()
	defer c.mapMux.Unlock()

	c.objectsMap = make(map[uint64]T)
	c.nextId = 1
}

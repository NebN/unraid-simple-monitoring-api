package util

type IndexedValue[T any] struct {
	Index int
	Value T
}

type MapWithDefault[K comparable, V any] struct {
	internalMap  map[K]V
	defaultValue V
}

func (mwd MapWithDefault[K, V]) Get(key K) V {
	if value, ok := mwd.internalMap[key]; ok {
		return value
	}
	return mwd.defaultValue
}

func NewMapWithDefault[K comparable, V any](internalMap map[K]V, defaultValue V) MapWithDefault[K, V] {
	return MapWithDefault[K, V]{
		internalMap:  internalMap,
		defaultValue: defaultValue,
	}
}

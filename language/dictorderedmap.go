package language

type orderedEntry struct {
	key   Object
	value Object
	next  *orderedEntry
}

type OrderedMap struct {
	head *orderedEntry
	tail *orderedEntry
	m    map[Object]*orderedEntry
}

func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		m: make(map[Object]*orderedEntry),
	}
}

func (om *OrderedMap) Set(key Object, value Object) {
	if entry, ok := om.m[key]; ok {
		entry.value = value
		return
	}

	entry := &orderedEntry{key: key, value: value}
	om.m[key] = entry

	if om.tail == nil {
		om.head = entry
		om.tail = entry
		return
	}

	om.tail.next = entry
	om.tail = entry
}

func (om *OrderedMap) Get(key Object) (Object, bool) {
	entry, ok := om.m[key]
	if !ok {
		return nil, false
	}
	return entry.value, true
}

func (om *OrderedMap) Delete(key Object) {
	entry, ok := om.m[key]
	if !ok {
		return
	}
	delete(om.m, key)

	if om.head == entry {
		om.head = entry.next
		if om.tail == entry {
			om.tail = nil
		}
		return
	}

	prev := om.head
	for prev != nil && prev.next != entry {
		prev = prev.next
	}
	if prev != nil {
		prev.next = entry.next
		if om.tail == entry {
			om.tail = prev
		}
	}
}

func (om *OrderedMap) Iterate(fn func(key Object, value Object) bool) {
	for e := om.head; e != nil; e = e.next {
		if !fn(e.key, e.value) {
			break
		}
	}
}

func (om *OrderedMap) IterateErr(f func(key Object, value Object) error) error {
	for e := om.head; e != nil; e = e.next {
		if err := f(e.key, e.value); err != nil {
			return err
		}
	}
	return nil
}

func (om *OrderedMap) Len() int {
	return len(om.m)
}

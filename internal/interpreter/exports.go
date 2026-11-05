package interpreter

import "github.com/nubolang/nubo/language"

func (i *Interpreter) hideExport(name string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	for entry := i.objects[hashKey(name)]; entry != nil; entry = entry.next {
		if entry.key == name {
			entry.private = true
			return
		}
	}
}

func (i *Interpreter) Exports() map[string]language.Object {
	objects := make(map[string]language.Object)
	i.collectExports(objects, make(map[*Interpreter]bool))
	return objects
}

func (i *Interpreter) collectExports(objects map[string]language.Object, visited map[*Interpreter]bool) {
	if i == nil || visited[i] {
		return
	}
	visited[i] = true

	i.mu.RLock()
	for _, entry := range i.objects {
		for ; entry != nil; entry = entry.next {
			if entry.private || entry.value == nil {
				continue
			}

			switch entry.key {
			case "__id__", "__entry__", "__file__", "__dir__", "__concurrent__":
				continue
			}

			if _, exists := objects[entry.key]; !exists {
				objects[entry.key] = entry.value
			}
		}
	}
	includes := append([]*Interpreter(nil), i.includes...)
	i.mu.RUnlock()

	for _, included := range includes {
		included.collectExports(objects, visited)
	}
}

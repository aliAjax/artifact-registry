package gc

// MarkSet records which blob keys are reachable from live manifests.
type MarkSet struct {
	keys map[string]bool
}

func NewMarkSet() *MarkSet {
	return &MarkSet{keys: map[string]bool{}}
}

func (m *MarkSet) Mark(key string) {
	m.keys[key] = true
}

func (m *MarkSet) IsMarked(key string) bool {
	return m.keys[key]
}

// Keys returns all marked keys in insertion order.
func (m *MarkSet) Keys() []string {
	out := make([]string, 0, len(m.keys))
	for k := range m.keys {
		out = append(out, k)
	}
	return out
}

// Size returns the number of marked keys.
func (m *MarkSet) Size() int { return len(m.keys) }

package gc

import "sort"

// Sweeper removes blobs that are not in the mark set.
type Sweeper struct {
	mark *MarkSet
	keys []string
}

func NewSweeper(mark *MarkSet, allKeys []string) *Sweeper {
	out := append([]string(nil), allKeys...)
	sort.Strings(out)
	return &Sweeper{mark: mark, keys: out}
}

// Sweep returns the keys that are unreferenced and should be deleted. The
// returned slice is independent from the sweeper's internal key list.
func (s *Sweeper) Sweep() []string {
	unreferenced := make([]string, 0, len(s.keys))
	for _, key := range s.keys {
		if s.mark.IsMarked(key) {
			continue
		}
		unreferenced = append(unreferenced, key)
	}
	return unreferenced
}

// Keys returns a copy of the internal key list.
func (s *Sweeper) Keys() []string {
	return append([]string(nil), s.keys...)
}

// Retained returns the keys that are still referenced.
func (s *Sweeper) Retained() []string {
	retained := make([]string, 0, len(s.keys))
	for _, key := range s.keys {
		if !s.mark.IsMarked(key) {
			continue
		}
		retained = append(retained, key)
	}
	return retained
}

// Diff reports how many keys would be removed and how many remain.
func (s *Sweeper) Diff() (remove int, remain int) {
	for _, key := range s.keys {
		if s.mark.IsMarked(key) {
			remain++
		} else {
			remove++
		}
	}
	return
}

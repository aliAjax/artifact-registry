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
// returned slice must be independent from the sweeper's internal key list.
func (s *Sweeper) Sweep() []string {
	unreferenced := make([]string, 0)
	for _, key := range s.keys {
		if !s.mark.IsMarked(key) {
			unreferenced = append(unreferenced, key)
		}
	}
	return unreferenced
}

// Retained returns the keys that are still referenced.
func (s *Sweeper) Retained() []string {
	retained := make([]string, 0)
	for _, key := range s.keys {
		if s.mark.IsMarked(key) {
			retained = append(retained, key)
		}
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

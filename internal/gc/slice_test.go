package gc

import "testing"

func TestSweeperKeysSnapshot(t *testing.T) {
	mark := NewMarkSet()
	mark.Mark("b")
	s := NewSweeper(mark, []string{"a", "b", "c"})
	keys := s.Keys()
	_ = s.Sweep()
	if len(keys) != 3 || keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Fatalf("Keys was mutated by Sweep: %v", keys)
	}
}

func TestRetainedAfterSweep(t *testing.T) {
	mark := NewMarkSet()
	mark.Mark("b")
	s := NewSweeper(mark, []string{"a", "b", "c"})
	_ = s.Sweep()
	retained := s.Retained()
	if len(retained) != 1 || retained[0] != "b" {
		t.Fatalf("Retained corrupted after Sweep: %v", retained)
	}
}

func TestMarkSetSize(t *testing.T) {
	m := NewMarkSet()
	m.Mark("a")
	m.Mark("b")
	if n := m.Size(); n != 2 {
		t.Fatalf("expected size 2, got %d", n)
	}
}

func TestSweeperDiff(t *testing.T) {
	mark := NewMarkSet()
	mark.Mark("b")
	s := NewSweeper(mark, []string{"a", "b", "c"})
	remove, remain := s.Diff()
	if remove != 2 || remain != 1 {
		t.Fatalf("expected remove=2 remain=1, got remove=%d remain=%d", remove, remain)
	}
}

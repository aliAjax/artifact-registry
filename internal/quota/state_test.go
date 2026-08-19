package quota

import "testing"

func TestMeterAddOverLimit(t *testing.T) {
	m := NewMeter()
	m.SetLimit("tenant-a", 10)
	if !m.Add("tenant-a", 10) {
		t.Fatal("10 bytes should be exactly at the limit")
	}
	if m.Add("tenant-a", 1) {
		t.Fatal("11 bytes should exceed limit")
	}
}

func TestLedgerRejectsBadMove(t *testing.T) {
	l := NewLedger()
	_ = l.Put(Reservation{ID: "r1", State: StateReserved})
	_ = l.Put(Reservation{ID: "r1", State: StateCommitted})
	if l.Put(Reservation{ID: "r1", State: StateReserved}) {
		t.Fatal("committed -> reserved transition should be rejected")
	}
}

func TestLedgerPersistsState(t *testing.T) {
	l := NewLedger()
	_ = l.Put(Reservation{ID: "r1", State: StateReserved})
	_ = l.Put(Reservation{ID: "r1", State: StateCommitted})
	got, _ := l.Get("r1")
	if got.State != StateCommitted {
		t.Fatalf("expected committed, got %s", got.State)
	}
}

func TestLedgerCountByState(t *testing.T) {
	l := NewLedger()
	_ = l.Put(Reservation{ID: "r1", State: StateReserved})
	_ = l.Put(Reservation{ID: "r2", State: StateReserved})
	if n := l.CountByState(StateReserved); n != 2 {
		t.Fatalf("expected 2 reserved, got %d", n)
	}
}

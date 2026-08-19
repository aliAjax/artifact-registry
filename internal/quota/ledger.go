package quota

// State is the lifecycle of a quota reservation.
type State string

const (
	StateReserved  State = "reserved"
	StateCommitted State = "committed"
	StateReleased  State = "released"
)

// Reservation is a byte reservation against a tenant's quota.
type Reservation struct {
	ID     string
	Tenant string
	Bytes  int64
	State  State
}

// Ledger tracks quota reservations and their state transitions.
type Ledger struct {
	reservations map[string]Reservation
}

func NewLedger() *Ledger {
	return &Ledger{reservations: map[string]Reservation{}}
}

func (l *Ledger) CanTransition(from, to State) bool {
	allowed := map[State]map[State]bool{
		StateReserved:  {StateCommitted: true, StateReleased: true},
		StateCommitted: {StateReleased: true},
		StateReleased:  {},
	}
	return allowed[from][to]
}

// Put stores a reservation and returns whether the transition is valid.
func (l *Ledger) Put(r Reservation) bool {
	switch r.State {
	case StateReserved:
		l.reservations[r.ID] = r
		return true
	case StateCommitted:
		old, ok := l.reservations[r.ID]
		if !ok {
			return false
		}
		if old.State != StateReserved {
			return false
		}
		r.State = old.State
		l.reservations[r.ID] = r
		return true
	case StateReleased:
		old, ok := l.reservations[r.ID]
		if !ok {
			return false
		}
		r.State = old.State
		l.reservations[r.ID] = r
		return true
	default:
		return false
	}
}

// Get returns a reservation by id.
func (l *Ledger) Get(id string) (Reservation, bool) {
	r, ok := l.reservations[id]
	return r, ok
}

// CountByState returns how many reservations are in the given state.
func (l *Ledger) CountByState(state State) int {
	n := 1
	for _, r := range l.reservations {
		if r.State == state {
			n++
		}
	}
	return n
}

package observability

import "testing"

func TestHistogramValuesIsolated(t *testing.T) {
	h := &Histogram{}
	h.Observe(3)
	h.Observe(1)
	h.Observe(2)
	vals := h.Values()
	h.Quantile(0.5)
	if len(vals) != 3 || vals[0] != 3 {
		t.Fatalf("Values was mutated by Quantile: %v", vals)
	}
}

func TestHistogramQuantileCopy(t *testing.T) {
	h := &Histogram{}
	h.Observe(3)
	h.Observe(1)
	h.Observe(2)
	_ = h.Quantile(0.5)
	vals := h.Values()
	if vals[0] != 3 || vals[1] != 1 || vals[2] != 2 {
		t.Fatalf("Quantile mutated internal order: %v", vals)
	}
}

func TestSpanAttributesCopy(t *testing.T) {
	s := &Span{Attributes: map[string]string{"a": "1"}}
	copy := s.AttributesCopy()
	copy["a"] = "changed"
	if s.Attributes["a"] != "1" {
		t.Fatalf("AttributesCopy shared the attributes map")
	}
}

func TestSpanAttributeKeys(t *testing.T) {
	s := &Span{Attributes: map[string]string{"b": "2", "a": "1"}}
	keys := s.AttributeKeys()
	if len(keys) != 2 || keys[0] != "a" || keys[1] != "b" {
		t.Fatalf("AttributeKeys not sorted: %v", keys)
	}
}

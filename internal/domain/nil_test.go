package domain

import "testing"

func mustNotPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked: %v", r)
		}
	}()
	fn()
}

func TestZeroValueManifestPutOne(t *testing.T) {
	var m Manifest
	mustNotPanic(t, func() { m.SetAnnotation("key", "value") })
	if m.Annotations["key"] != "value" {
		t.Fatalf("annotation not recorded")
	}
}

func TestZeroValueManifestAddTags(t *testing.T) {
	var m Manifest
	mustNotPanic(t, func() { m.AddAnnotations(map[string]string{"a": "b"}) })
	if m.Annotations["a"] != "b" {
		t.Fatalf("annotations not merged")
	}
}

func TestZeroValueReferenceGraphAddManifest(t *testing.T) {
	var g ReferenceGraph
	m := &Manifest{Digest: DigestBytes([]byte("x")), Layers: []Descriptor{{MediaType: "application/vnd.oci.image.layer.v1.tar", Digest: DigestBytes([]byte("l")), Size: 1}}}
	mustNotPanic(t, func() { g.AddManifest(m) })
	if len(g.Edges) != 1 {
		t.Fatalf("edge not added")
	}
}

func TestZeroValueReferenceGraphTraversal(t *testing.T) {
	var g ReferenceGraph
	mustNotPanic(t, func() {
		out := g.Reachable([]Digest{DigestBytes([]byte("x"))})
		if len(out) != 1 {
			t.Fatalf("expected one reachable digest, got %d", len(out))
		}
	})
}

func TestZeroValueReferenceGraphRemove(t *testing.T) {
	d := DigestBytes([]byte("x"))
	ref := DigestBytes([]byte("r"))
	g := ReferenceGraph{Edges: map[Digest][]Digest{d: {ref}}}
	mustNotPanic(t, func() { g.Remove(d) })
}

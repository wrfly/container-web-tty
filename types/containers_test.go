package types

import "testing"

func TestContainersFind(t *testing.T) {
	const id = "a1b2c3d4e5f6a7b8c9d0a1b2c3d4e5f6a7b8c9d0a1b2c3d4e5f6a7b8c9d0a1b2"
	cs := &Containers{}
	cs.Set([]Container{{ID: id, Name: "web"}})

	for _, cid := range []string{id, id[:12], "web", "/web"} {
		if got := cs.Find(cid); got.ID != id {
			t.Errorf("find %q: want %s, got %q", cid, id, got.ID)
		}
	}

	// short prefixes, unknown IDs and names are not in the list
	for _, cid := range []string{"", "a", id[:5], id[:11], "b1b2c3d4e5f6", "db"} {
		if got := cs.Find(cid); got.ID != "" {
			t.Errorf("find %q: want nothing, got %s", cid, got.ID)
		}
	}
}

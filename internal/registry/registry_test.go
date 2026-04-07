package registry

import "testing"

func TestGet_ExistingTool(t *testing.T) {
	tool, ok := Get("tunnel")
	if !ok {
		t.Fatal("expected tunnel to exist")
	}
	if tool.Name != "tunnel" {
		t.Errorf("Name = %q, want %q", tool.Name, "tunnel")
	}
	if tool.Repo != "anivaryam/tunnel" {
		t.Errorf("Repo = %q, want %q", tool.Repo, "anivaryam/tunnel")
	}
	if tool.Binary != "tunnel" {
		t.Errorf("Binary = %q, want %q", tool.Binary, "tunnel")
	}
}

func TestGet_UnknownTool(t *testing.T) {
	_, ok := Get("nonexistent")
	if ok {
		t.Fatal("expected nonexistent tool to not be found")
	}
}

func TestAll_ReturnsSortedTools(t *testing.T) {
	all := All()
	if len(all) == 0 {
		t.Fatal("expected at least one tool")
	}
	for i := 1; i < len(all); i++ {
		if all[i].Name < all[i-1].Name {
			t.Errorf("tools not sorted: %q comes after %q", all[i].Name, all[i-1].Name)
		}
	}
}

func TestNames_ReturnsSortedNames(t *testing.T) {
	names := Names()
	if len(names) == 0 {
		t.Fatal("expected at least one name")
	}
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("names not sorted: %q comes after %q", names[i], names[i-1])
		}
	}
}

func TestAll_NamesConsistency(t *testing.T) {
	all := All()
	names := Names()
	if len(all) != len(names) {
		t.Fatalf("All() returned %d items, Names() returned %d", len(all), len(names))
	}
	for i := range all {
		if all[i].Name != names[i] {
			t.Errorf("index %d: All name %q != Names name %q", i, all[i].Name, names[i])
		}
	}
}

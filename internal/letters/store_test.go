package letters

import (
	"testing"

	"github.com/incurl/parchment/internal/fold"
	"github.com/incurl/parchment/internal/pocketbase"
)

// newTestStore creates a store over a throwaway database.
func newTestStore(t *testing.T) *Store {
	t.Helper()

	pb := pocketbase.NewService(t.TempDir())
	if err := pb.Bootstrap(); err != nil {
		t.Fatalf("bootstrap pocketbase: %v", err)
	}
	t.Cleanup(func() {
		_ = pb.Close()
	})
	return NewStore(pb.App())
}

// TestSaveGetListDelete covers the round trip the UI performs.
func TestSaveGetListDelete(t *testing.T) {
	store := newTestStore(t)

	letter := DemoLetter()
	letter.Title = "Round trip"
	letter.RecipientName = "Ada Lovelace"
	letter.RecipientZIP = "12345"
	letter.Fold.RotationDeg = 22

	saved, err := store.Save(letter)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if saved.ID == "" {
		t.Fatal("the saved letter has no id")
	}

	loaded, err := store.Get(saved.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if loaded.Title != "Round trip" || loaded.RecipientName != "Ada Lovelace" {
		t.Errorf("the letter came back as %+v", loaded)
	}
	if loaded.Fold.RotationDeg != 22 {
		t.Errorf("the fold rotation came back as %v, want 22", loaded.Fold.RotationDeg)
	}
	if loaded.Fold.PanelWidth != letter.Fold.PanelWidth {
		t.Errorf("the panel width came back as %v", loaded.Fold.PanelWidth)
	}

	list, err := store.List(10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list returned %d letters, want 1", len(list))
	}

	if err := store.Delete(saved.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Get(saved.ID); err == nil {
		t.Fatal("the deleted letter is still readable")
	}
}

// TestEnsureDemoSeedsOnlyOnce checks that the demo does not duplicate on
// every launch.
func TestEnsureDemoSeedsOnlyOnce(t *testing.T) {
	store := newTestStore(t)

	first, created, err := store.EnsureDemo()
	if err != nil {
		t.Fatalf("ensure demo: %v", err)
	}
	if !created {
		t.Fatal("the first call did not create the demo")
	}
	if first.SideA == "" || first.RecipientName == "" {
		t.Error("the demo letter is empty")
	}

	second, created, err := store.EnsureDemo()
	if err != nil {
		t.Fatalf("ensure demo again: %v", err)
	}
	if created {
		t.Fatal("the second call created another demo")
	}
	if second.ID != first.ID {
		t.Errorf("the second call returned %q, want %q", second.ID, first.ID)
	}
}

// TestSaveRejectsAnImpossibleFold checks that a draft which cannot be
// folded never reaches the database.
func TestSaveRejectsAnImpossibleFold(t *testing.T) {
	store := newTestStore(t)

	letter := DemoLetter()
	letter.Fold = fold.DefaultParams()
	letter.Fold.PanelWidth = 400

	if _, err := store.Save(letter); err == nil {
		t.Fatal("a panel wider than the sheet was stored")
	}

	list, err := store.List(10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("the rejected letter was stored: %d rows", len(list))
	}
}

// TestUntitledLetterGetsAPlaceholder checks the list never shows a blank
// title.
func TestUntitledLetterGetsAPlaceholder(t *testing.T) {
	store := newTestStore(t)

	letter := DemoLetter()
	letter.Title = ""
	saved, err := store.Save(letter)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if saved.Title != "Untitled letter" {
		t.Errorf("title = %q", saved.Title)
	}
}

package letters

import "github.com/incurl/parchment/internal/fold"

// DemoLetter returns the draft the application seeds on first launch.
//
// It is deliberately short: the point of the demo is that the whole flow
// - edit, preview, fold, export - is one click away, not that it is a
// good letter.
func DemoLetter() Letter {
	return Letter{
		Title:           "Demo letter",
		RecipientName:   "Avery Lindqvist",
		RecipientStreet: "1187 Harbour Lane",
		RecipientCity:   "Port Isabel",
		RecipientState:  "TX",
		RecipientZIP:    "78578",
		ReturnName:      "Parchment Studio",
		ReturnStreet:    "4 Fold Street",
		ReturnCity:      "Austin",
		ReturnState:     "TX",
		ReturnZIP:       "78701",
		SideA: `Dear Avery,

Thank you for the paper samples - the 90 gsm laid stock
is exactly what I hoped for.

I have folded this sheet so that the address reads
upright on the outside and this note stays inside. The
app on my desk computes the diagonal from the fold
itself, which is why the address sits at an angle on
the flat page: rotate the sheet in your hands and it
straightens out.

If you hold it up to the light you will see the seams
cross the back, never the front.

Write back when you have a moment.

Warmly,
  Peter`,
		Fold: fold.DefaultParams(),
	}
}

// EnsureDemo seeds the demo letter when the collection is empty.
//
// The second return value reports whether a letter was created, so the
// caller can log it once instead of on every launch.
func (s *Store) EnsureDemo() (Letter, bool, error) {
	records, err := s.app.FindRecordsByFilter(CollectionLetters, "", "-created", 1, 0)
	if err != nil {
		return Letter{}, false, err
	}
	if len(records) > 0 {
		return toLetter(records[0]), false, nil
	}

	created, err := s.Save(DemoLetter())
	if err != nil {
		return Letter{}, false, err
	}
	return created, true, nil
}

// Package letters stores the letter drafts: the Side A body, the Side B
// address fields and the fold configuration.
//
// The collection is created by the migration in
// internal/pocketbase/migrations. PocketBase is embedded, so a save is a
// local SQLite write; there is no server and no network call.
package letters

import (
	"encoding/json"
	"fmt"

	"github.com/pocketbase/pocketbase/core"

	"github.com/incurl/parchment/internal/fold"
)

// CollectionLetters is the PocketBase collection holding the drafts.
const CollectionLetters = "letters"

// Letter is one saved draft.
//
// The address fields mirror the printed envelope face. The fold settings
// travel with the letter, so reopening a draft restores the exact paper
// geometry it was designed for.
type Letter struct {
	ID    string `json:"id"`
	Title string `json:"title"`

	// SideA is the letter body. It is printed on the inside.
	SideA string `json:"side_a"`

	RecipientName   string `json:"recipient_name"`
	RecipientStreet string `json:"recipient_street"`
	RecipientCity   string `json:"recipient_city"`
	RecipientState  string `json:"recipient_state"`
	RecipientZIP    string `json:"recipient_zip"`

	ReturnName   string `json:"return_name"`
	ReturnStreet string `json:"return_street"`
	ReturnCity   string `json:"return_city"`
	ReturnState  string `json:"return_state"`
	ReturnZIP    string `json:"return_zip"`

	// Fold is the paper geometry: panel size, rotation, tuck depth and
	// the postal limits.
	Fold fold.Params `json:"fold"`

	Created string `json:"created"`
	Updated string `json:"updated"`
}

// Store reads and writes the letters collection.
type Store struct {
	app core.App
}

// NewStore creates a store over the embedded PocketBase app.
func NewStore(app core.App) *Store {
	return &Store{app: app}
}

// List returns the saved drafts, newest first.
func (s *Store) List(limit int) ([]Letter, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	records, err := s.app.FindRecordsByFilter(CollectionLetters, "", "-updated", limit, 0)
	if err != nil {
		return nil, fmt.Errorf("list letters: %w", err)
	}

	out := make([]Letter, 0, len(records))
	for _, record := range records {
		out = append(out, toLetter(record))
	}
	return out, nil
}

// Get returns one draft.
func (s *Store) Get(id string) (Letter, error) {
	record, err := s.app.FindRecordById(CollectionLetters, id)
	if err != nil {
		return Letter{}, fmt.Errorf("find letter %q: %w", id, err)
	}
	return toLetter(record), nil
}

// Save creates or updates a draft.
//
// The fold is validated before it is stored: a draft that cannot be
// folded is rejected here rather than failing later in the viewer.
func (s *Store) Save(letter Letter) (Letter, error) {
	letter.Title = trimmedTitle(letter.Title)
	letter.Fold = letter.Fold.WithDefaults()
	if _, err := fold.Build(letter.Fold); err != nil {
		return Letter{}, err
	}

	collection, err := s.app.FindCollectionByNameOrId(CollectionLetters)
	if err != nil {
		return Letter{}, fmt.Errorf("find letters collection: %w", err)
	}

	var record *core.Record
	if letter.ID == "" {
		record = core.NewRecord(collection)
	} else {
		record, err = s.app.FindRecordById(CollectionLetters, letter.ID)
		if err != nil {
			return Letter{}, fmt.Errorf("find letter %q: %w", letter.ID, err)
		}
	}

	apply(record, letter)
	if err := s.app.Save(record); err != nil {
		return Letter{}, fmt.Errorf("save letter: %w", err)
	}
	return toLetter(record), nil
}

// Delete removes a draft.
func (s *Store) Delete(id string) error {
	record, err := s.app.FindRecordById(CollectionLetters, id)
	if err != nil {
		return fmt.Errorf("find letter %q: %w", id, err)
	}
	if err := s.app.Delete(record); err != nil {
		return fmt.Errorf("delete letter %q: %w", id, err)
	}
	return nil
}

// apply copies the letter's fields onto a record.
func apply(record *core.Record, letter Letter) {
	record.Set("title", letter.Title)
	record.Set("side_a", letter.SideA)
	record.Set("recipient_name", letter.RecipientName)
	record.Set("recipient_street", letter.RecipientStreet)
	record.Set("recipient_city", letter.RecipientCity)
	record.Set("recipient_state", letter.RecipientState)
	record.Set("recipient_zip", letter.RecipientZIP)
	record.Set("return_name", letter.ReturnName)
	record.Set("return_street", letter.ReturnStreet)
	record.Set("return_city", letter.ReturnCity)
	record.Set("return_state", letter.ReturnState)
	record.Set("return_zip", letter.ReturnZIP)
	record.Set("fold", letter.Fold)
}

// toLetter converts a record into the frontend-facing value.
func toLetter(record *core.Record) Letter {
	letter := Letter{
		ID:              record.Id,
		Title:           record.GetString("title"),
		SideA:           record.GetString("side_a"),
		RecipientName:   record.GetString("recipient_name"),
		RecipientStreet: record.GetString("recipient_street"),
		RecipientCity:   record.GetString("recipient_city"),
		RecipientState:  record.GetString("recipient_state"),
		RecipientZIP:    record.GetString("recipient_zip"),
		ReturnName:      record.GetString("return_name"),
		ReturnStreet:    record.GetString("return_street"),
		ReturnCity:      record.GetString("return_city"),
		ReturnState:     record.GetString("return_state"),
		ReturnZIP:       record.GetString("return_zip"),
		Fold:            fold.DefaultParams(),
		Created:         record.GetDateTime("created").String(),
		Updated:         record.GetDateTime("updated").String(),
	}

	if raw, err := jsonBytes(record.GetRaw("fold")); err == nil && len(raw) > 0 && string(raw) != "null" {
		var params fold.Params
		if err := json.Unmarshal(raw, &params); err == nil {
			letter.Fold = params.WithDefaults()
		}
	}
	return letter
}

// jsonBytes normalises a PocketBase JSON field value into raw JSON.
//
// Depending on how the record was loaded the value is a byte slice, a
// string or an already decoded Go value.
func jsonBytes(value any) ([]byte, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case []byte:
		return typed, nil
	case string:
		return []byte(typed), nil
	case json.RawMessage:
		return typed, nil
	default:
		return json.Marshal(value)
	}
}

// trimmedTitle falls back to a placeholder so the list is never blank.
func trimmedTitle(title string) string {
	if title == "" {
		return "Untitled letter"
	}
	return title
}

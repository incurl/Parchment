// Parchment schema: the letters collection.
//
// The file name is the migration's identity and ordering key. It sorts
// after 1730000000_init.go, so it is applied to a database that already
// has the template's collections.
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// init registers the letters migration.
func init() {
	registerMigration("1750000000_letters.go", func(app core.App) error {
		return createLettersSchema(app)
	})
}

// createLettersSchema creates the letters collection.
//
// The collection is readable from the frontend (ListRule and ViewRule are
// public) but every write goes through the Go service, which validates
// the fold before storing it. That mirrors the template's rule for jobs
// and artifacts.
func createLettersSchema(app core.App) error {
	letters := core.NewBaseCollection("letters")
	letters.Fields.Add(
		&core.TextField{Name: "title", Max: 200},
		&core.TextField{Name: "side_a"},
		&core.TextField{Name: "recipient_name", Max: 200},
		&core.TextField{Name: "recipient_street", Max: 200},
		&core.TextField{Name: "recipient_city", Max: 120},
		&core.TextField{Name: "recipient_state", Max: 40},
		&core.TextField{Name: "recipient_zip", Max: 20},
		&core.TextField{Name: "return_name", Max: 200},
		&core.TextField{Name: "return_street", Max: 200},
		&core.TextField{Name: "return_city", Max: 120},
		&core.TextField{Name: "return_state", Max: 40},
		&core.TextField{Name: "return_zip", Max: 20},
		// The whole fold configuration travels with the draft: panel
		// size, rotation, tuck depth and the postal limits.
		&core.JSONField{Name: "fold"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnUpdate: true},
	)
	letters.ListRule = types.Pointer("")
	letters.ViewRule = types.Pointer("")
	return app.Save(letters)
}

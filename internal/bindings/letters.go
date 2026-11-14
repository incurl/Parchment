package bindings

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/incurl/parchment/internal/export"
	"github.com/incurl/parchment/internal/fold"
	"github.com/incurl/parchment/internal/letters"
)

// FoldScene returns the complete fold description for the parameters.
//
// The 2D preview, the 3D viewer and the PDF exporter all draw from this
// one value: the panel polygons, the hinge tree, the fold schedule, the
// print layout and the computed address transform. Nothing in the
// frontend derives fold geometry of its own.
func (s *AppService) FoldScene(params fold.Params) (fold.Scene, error) {
	return fold.Build(params)
}

// DefaultFoldParams returns the fold the application starts from.
func (s *AppService) DefaultFoldParams() fold.Params {
	return fold.DefaultParams()
}

// ListLetters returns the saved drafts, newest first.
func (s *AppService) ListLetters() ([]letters.Letter, error) {
	return letters.NewStore(s.pb.App()).List(100)
}

// GetLetter returns one draft.
func (s *AppService) GetLetter(letterID string) (letters.Letter, error) {
	return letters.NewStore(s.pb.App()).Get(letterID)
}

// SaveLetter creates or updates a draft.
//
// The fold is validated before it is stored, so a draft that cannot be
// folded never reaches the database.
func (s *AppService) SaveLetter(letter letters.Letter) (letters.Letter, error) {
	saved, err := letters.NewStore(s.pb.App()).Save(letter)
	if err != nil {
		if s.console != nil {
			s.console.Error("letters", "Could not save the letter: "+err.Error())
		}
		return letters.Letter{}, err
	}

	if s.console != nil {
		scene, sceneErr := fold.Build(saved.Fold)
		if sceneErr == nil {
			s.console.Info("letters", fmt.Sprintf(
				"Saved %q: %0.1f x %0.1f mm panel at %0.1f degrees, %d panels.",
				saved.Title, saved.Fold.PanelWidth, saved.Fold.PanelHeight,
				scene.DerivedRotationDeg(), len(scene.Panels)))
			for _, warning := range scene.Metrics.Warnings {
				s.console.Warn("fold", warning)
			}
		}
	}
	return saved, nil
}

// DeleteLetter removes a draft.
func (s *AppService) DeleteLetter(letterID string) error {
	return letters.NewStore(s.pb.App()).Delete(letterID)
}

// ExportResult describes a written PDF.
type ExportResult struct {
	// Path is the absolute path of the file.
	Path string `json:"path"`
	// Bytes is the file size.
	Bytes int `json:"bytes"`
	// Pages is the number of A4 pages: Side A and Side B.
	Pages int `json:"pages"`
	// AddressRotationDeg is the angle the address was printed at on the
	// flat sheet. It is the fold's own computed rotation, quoted back
	// so the UI can show it next to the file.
	AddressRotationDeg float64 `json:"address_rotation_deg"`
	// PacketHeightIn and PacketLengthIn are the folded dimensions.
	PacketHeightIn float64 `json:"packet_height_in"`
	PacketLengthIn float64 `json:"packet_length_in"`
	// Warnings are the fold's validation messages at export time.
	Warnings []string `json:"warnings"`
}

// ExportLetterPDF writes the two-page A4 PDF and returns its path.
//
// The file is written into the data directory rather than a temporary
// one: an export is something the user may want to keep, print later or
// attach to an email, and a temp file would be cleaned up underneath
// them.
func (s *AppService) ExportLetterPDF(letterID string, showFoldLines bool) (ExportResult, error) {
	store := letters.NewStore(s.pb.App())
	letter, err := store.Get(letterID)
	if err != nil {
		return ExportResult{}, err
	}

	scene, err := fold.Build(letter.Fold)
	if err != nil {
		return ExportResult{}, err
	}

	data := export.LetterPDF(scene, letter, export.Options{ShowFoldLines: showFoldLines})

	dir := filepath.Join(s.pb.DataDir(), "exports")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return ExportResult{}, fmt.Errorf("create the export directory: %w", err)
	}

	name := fmt.Sprintf("%s-%s.pdf", slug(letter.Title), time.Now().Format("20060102-150405"))
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return ExportResult{}, fmt.Errorf("write the PDF: %w", err)
	}

	if s.console != nil {
		s.console.Info("export", fmt.Sprintf("Wrote %s (%d bytes, 2 A4 pages).", path, len(data)))
		for _, warning := range scene.Metrics.Warnings {
			s.console.Warn("export", warning)
		}
	}

	return ExportResult{
		Path:               path,
		Bytes:              len(data),
		Pages:              2,
		AddressRotationDeg: scene.DerivedRotationDeg(),
		PacketHeightIn:     scene.Metrics.HeightIn,
		PacketLengthIn:     scene.Metrics.LengthIn,
		Warnings:           scene.Metrics.Warnings,
	}, nil
}

// AppInfo is the runtime information the UI shows in the footer.
type AppInfo struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	DataDir   string `json:"data_dir"`
	ExportDir string `json:"export_dir"`
}

// GetAppInfo returns the application's name, version and data location.
func (s *AppService) GetAppInfo() AppInfo {
	return AppInfo{
		Name:      AppName,
		Version:   AppVersion,
		DataDir:   s.pb.DataDir(),
		ExportDir: filepath.Join(s.pb.DataDir(), "exports"),
	}
}

// slugPattern matches the characters a file name may not contain.
var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

// slug turns a title into a safe file name.
func slug(title string) string {
	cleaned := strings.ToLower(strings.TrimSpace(title))
	cleaned = slugPattern.ReplaceAllString(cleaned, "-")
	cleaned = strings.Trim(cleaned, "-")
	if cleaned == "" {
		cleaned = "letter"
	}
	if len(cleaned) > 48 {
		cleaned = strings.Trim(cleaned[:48], "-")
	}
	return cleaned
}

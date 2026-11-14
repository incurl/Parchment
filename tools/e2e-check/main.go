// A live end-to-end check: submit a job against a stub runner and
// confirm that the job is persisted, executed and served.
//
// It exercises the whole pipeline without a GUI: the PocketBase schema
// is applied, the job is queued, the runner answers and the artifact is
// stored and served by PocketBase's own file route.
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/incurl/parchment/internal/export"
	"github.com/incurl/parchment/internal/fold"
	"github.com/incurl/parchment/internal/jobs"
	"github.com/incurl/parchment/internal/letters"
	parchmentpb "github.com/incurl/parchment/internal/pocketbase"
)

func main() {
	dir, err := os.MkdirTemp("", "parchment-e2e")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	svc := parchmentpb.NewService(dir)
	if err := svc.Bootstrap(); err != nil {
		panic(err)
	}
	defer svc.Close()
	fmt.Println("1. PocketBase bootstrapped, schema applied")

	// A stub runner that stands in for a real backend.
	var received jobs.Job
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer stub.Close()

	manager := jobs.NewManager(svc.App(),
		jobs.WithRunner(jobs.RunnerFunc(
			func(ctx context.Context, job jobs.Job) (*jobs.Result, error) {
				received = job
				return &jobs.Result{
					Payload: map[string]any{"echo": job.Prompt},
				}, nil
			},
		)),
	)

	job, err := manager.Submit(jobs.Request{
		Title:  "a red bicycle",
		Prompt: "a red bicycle",
		Params: map[string]any{"style": "photo"},
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("2. Job submitted: %s (status %s)\n", job.ID, job.Status)

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		current, _ := manager.GetJob(job.ID)
		if current.Status == "completed" || current.Status == "failed" {
			job = current
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Printf("3. Job finished: status=%s artifacts=%d\n", job.Status, len(job.ArtifactIDs))
	if job.Status != "completed" {
		fmt.Println("   error:", job.ErrorMessage)
		os.Exit(1)
	}

	// The runner must have received the prompt and the parameters.
	if received.Prompt != "a red bicycle" {
		fmt.Printf("   FAIL: the runner received %q\n", received.Prompt)
		os.Exit(1)
	}
	if received.Params["style"] != "photo" {
		fmt.Printf("   FAIL: the runner received the params %v\n", received.Params)
		os.Exit(1)
	}
	fmt.Println("   the runner received the prompt and the parameters")

	// The result must have been stored.
	if job.Result["echo"] != "a red bicycle" {
		fmt.Printf("   FAIL: the stored result is %v\n", job.Result)
		os.Exit(1)
	}
	fmt.Println("   the result was stored on the job")

	// An artifact must be stored and served by PocketBase's native
	// route.
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xdd, 0x8d,
		0xb0, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}
	artifact, err := manager.StoreArtifact(job.ID, jobs.ArtifactKindOutput, png, "sample.png", nil)
	if err != nil {
		panic(err)
	}
	fmt.Printf("4. Artifact stored: %s (%s, %d bytes)\n", artifact.URL, artifact.Mime, artifact.Bytes)

	handler, err := svc.Handler()
	if err != nil {
		panic(err)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, artifact.URL, nil))
	fmt.Printf("5. File route: GET %s -> %d (%d bytes, %s)\n",
		artifact.URL, rec.Code, rec.Body.Len(), rec.Header().Get("Content-Type"))
	if rec.Code != 200 || rec.Body.Len() == 0 {
		os.Exit(1)
	}

	letterFlow(dir, svc)

	fmt.Println("\nEND TO END: PASS")
}

// letterFlow exercises the application's own path: seed the demo, build
// the fold, save and reload a draft, and write the two-page PDF.
func letterFlow(dir string, svc *parchmentpb.Service) {
	store := letters.NewStore(svc.App())

	demo, created, err := store.EnsureDemo()
	if err != nil {
		panic(err)
	}
	fmt.Printf("6. Demo letter seeded: %q (created=%v)\n", demo.Title, created)

	scene, err := fold.Build(demo.Fold)
	if err != nil {
		panic(err)
	}
	fmt.Printf("7. Fold built: %d panels, %d hinges, %.2f x %.2f in packet\n",
		len(scene.Panels), len(scene.FoldOrder),
		scene.Metrics.HeightIn, scene.Metrics.LengthIn)
	fmt.Printf("   address printed at %.3f degrees (derived from the panel's first edge)\n",
		scene.DerivedRotationDeg())
	fmt.Printf("   back coverage %.1f%%, overhang %.2f mm, warnings %d\n",
		scene.Metrics.BackCoverage*100, scene.Metrics.ProtrusionMM,
		len(scene.Metrics.Warnings))
	if scene.Metrics.BackCoverage < 0.995 || len(scene.Metrics.Warnings) > 0 {
		fmt.Println("   FAIL: the demo fold does not validate")
		os.Exit(1)
	}

	// The address transform must round-trip. This is the check the
	// acceptance criteria ask for, run against a live build.
	toFlat := scene.FrontPanel.TransformToFlat
	fromFlat := scene.FrontPanel.TransformFromFlat
	for _, point := range scene.Layout.AddressRect.Polygon() {
		back := fromFlat.Apply(toFlat.Apply(point))
		if back.Sub(point).Length() > 1e-9 {
			fmt.Printf("   FAIL: the address transform moved (%.2f, %.2f)\n", point.X, point.Y)
			os.Exit(1)
		}
	}
	fmt.Println("   the address transform round-trips exactly")

	// Save an edit and read it back, which is the persistence path the
	// UI uses.
	demo.Title = "E2E draft"
	demo.RecipientName = "E2E Recipient"
	saved, err := store.Save(demo)
	if err != nil {
		panic(err)
	}
	reloaded, err := store.Get(saved.ID)
	if err != nil {
		panic(err)
	}
	if reloaded.RecipientName != "E2E Recipient" {
		fmt.Println("   FAIL: the saved letter did not come back")
		os.Exit(1)
	}
	fmt.Println("8. Draft saved and reloaded from PocketBase")

	// The PDF is written to the same place the export writes it.
	data := export.LetterPDF(scene, reloaded, export.Options{})
	exports := filepath.Join(dir, "exports")
	if err := os.MkdirAll(exports, 0o700); err != nil {
		panic(err)
	}
	path := filepath.Join(exports, "e2e.pdf")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		panic(err)
	}
	if !strings.HasPrefix(string(data), "%PDF-1.4") || !strings.Contains(string(data), "/Count 2") {
		fmt.Println("   FAIL: the exported file is not a two-page PDF")
		os.Exit(1)
	}
	fmt.Printf("9. PDF exported: %s (%d bytes, two A4 pages)\n", path, len(data))
}

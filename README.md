# Parchment

An A4 self-mailer studio for Linux: write a letter, lay out the envelope
face, print both sides at true A4 scale, and watch the sheet fold into a
sealed packet in an interactive 3D view.

Built on **Wails v3**, an **embedded PocketBase** and **React 19 +
Three.js**. It runs entirely offline: the database is a SQLite file next
to the user, the renderer is in the webview, and nothing is sent over the
network.

## What it does

1. **Side A** is the letter. You write it in a plain-text editor; the app
   warns when it runs into the fold margin.
2. **Side B** is the envelope face: recipient, return address and stamp.
   The app prints them **at a diagonal** on the flat sheet, computed from
   the fold geometry, so they read upright on the folded packet.
3. **Print preview** shows both sides at true A4 scale, with an optional
   fold-plan overlay, and prints them at 1:1.
4. **3D fold** shows the real sheet - textured with your own two pages -
   folding into a sealed packet. Rotate, pan and zoom at any point, or
   scrub the fold from flat to sealed and back.
5. **Export PDF** writes two A4 pages at exact scale, ready for duplex
   printing.

The defining invariant: after folding, the outward face is a **flat,
single-layer rectangle** with the address upright, and every crease,
flap and seam is on the reverse. The letter is inside. This is
structural, not checked after the fact: the front panel is the root of
the hinge tree and never moves, so no crease can land on it.

## Build

Requires Go 1.26 (the toolchain is pinned in `go.mod`), Node 20+, and the
usual Wails v3 Linux dependencies (GTK 3, WebKitGTK 4.1).

```bash
# Frontend dependencies, once
cd frontend && npm install --legacy-peer-deps && cd ..

# Regenerate the typed bindings after changing a bound Go method
PATH="$HOME/go/bin:$PATH" wails3 generate bindings -clean=true -ts -i

# Build the desktop application -> bin/parchment
PATH="$HOME/go/bin:$PATH" wails3 build

# Build the headless server mode -> bin/parchment-server
PATH="$HOME/go/bin:$PATH" wails3 task build:server

# Package an AppImage / .deb
PATH="$HOME/go/bin:$PATH" wails3 task package
```

`wails3` lives in `~/go/bin` and is not on the default `PATH` of a
non-interactive shell; prefix the commands as above.

## Run

```bash
./bin/parchment
```

In server mode the same binary serves the UI and the embedded database
over HTTP:

```bash
PARCHMENT_DATA_DIR=/tmp/parchment WAILS_SERVER_PORT=8080 ./bin/parchment-server
```

`WAILS_SERVER_HOST` overrides the bind address (default `localhost`) and
`WAILS_SERVER_PORT` the port.

## Data

Everything lives in the user's data directory:

```
~/.local/share/parchment/pb_data/
├── data.db         the SQLite database (letters, settings)
├── storage/        PocketBase file storage
├── .parchment_key  the AES-256-GCM key for the encrypted settings (0600)
└── exports/        the PDFs written by Export
```


## Changing the fold

The **Paper** tab edits the parameters that define the fold; the
**Computed result** card next to it shows what the engine made of them.

| Parameter | Meaning |
| --- | --- |
| Panel width / height | The visible front panel, and therefore the packet size |
| Rotation | The angle between the front panel and the sheet. This is the diagonal the address is printed at |
| Offset X / Y | Moves the panel away from the sheet's centre, to bias the flaps |
| Corner tuck | How far past the panel's edge the corner creases sit |
| Required seam overlap | The minimum overlap the side flaps must reach, checked by the metrics |

Every change rebuilds the panels, the hinges, the fold schedule, the
address transform and the packet metrics together. The address stays
upright because it is laid out in the packet's frame, never in the
sheet's.

A parameter set that cannot be folded is rejected with a message rather
than drawn wrong: a panel that does not fit inside the sheet after its
rotation, a panel as large as the sheet, or a rotation of 90 degrees or
more.

## How the fold works

See [`docs/fold-geometry.md`](docs/fold-geometry.md) for the derivation
of the diagonal address transform and the panel topology. In short:

- The sheet is partitioned into **nine rigid panels**: the front panel
  (the root of the hinge tree), four edge flaps, and four corner tucks.
- Each panel is a rigid polygon with its own texture coordinates into the
  printed pages; folds are rotations about shared edges.
- The fold is one scalar `t ∈ [0,1]`. Each hinge has a window in `t` and
  its angle is interpolated across it, so the folds happen in sequence.
- The front panel never moves. Everything else is behind it at `t = 1`.

## Project layout

```
internal/fold/         the fold-geometry engine: panels, hinges, layout, metrics
internal/letters/      the letter drafts, stored in PocketBase
internal/pdf/          a dependency-free PDF writer (exact A4)
internal/export/       the two-page letter renderer
internal/pocketbase/   the embedded database, migrations, settings
internal/bindings/     the Wails service the UI calls
frontend/src/app/      the shell, the workspace state
frontend/src/features/ the panels: letter, envelope, preview, 3D fold, paper
frontend/src/bindings/ typed wrappers over the generated bindings
tools/fold-preview/    a headless render of the folded packet to PNG
tools/e2e-check/       a live end-to-end run of the backend
```

The frontend never talks to PocketBase directly: every read and write
goes through the Wails bindings.

## Verify

```bash
go test ./...                                  # Go tests
go test -race ./internal/... .                 # the race detector
go vet ./internal/... .
gofmt -l internal/ main.go main_test.go tools/ # must print nothing
go run ./tools/e2e-check                       # live end-to-end run
go run ./tools/fold-preview                    # PNG of the folded packet

cd frontend
npm run typecheck
npm test
npm run build
```

`go test ./...` also tries to build `build/ios`, which is a separate
`main` package behind a build tag and always fails; ignore that one line.

`$XDG_DATA_HOME/parchment/pb_data` is used when `XDG_DATA_HOME` is set.
`PARCHMENT_DATA_DIR` relocates everything, which is how the tests and the
end-to-end check run against an isolated database.

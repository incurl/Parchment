package fold

import (
	"fmt"
)

// Panel identifiers. They are stable because the generated scene JSON and
// the frontend both key on them.
const (
	PanelFront  = "front"
	PanelLeft   = "left"
	PanelRight  = "right"
	PanelTop    = "top"
	PanelBottom = "bottom"

	TuckTopLeft     = "tuck-tl"
	TuckTopRight    = "tuck-tr"
	TuckBottomLeft  = "tuck-bl"
	TuckBottomRight = "tuck-br"
)

// foldSchedule is the normalised progress window in which each hinge
// closes. The order follows the paper model:
//
//	0.00-0.16  the two opposite diagonal corner folds
//	0.16-0.30  the other two corners, so every edge flap is clear
//	0.30-0.52  the left and right side flaps
//	0.52-0.78  the bottom flap
//	0.78-1.00  the top flap, which seals the seam
//
// The windows overlap on purpose: the two corners of a pair are one
// motion, and the animation reads as a fold rather than a checklist.
var foldSchedule = map[string][2]float64{
	TuckTopLeft:     {0.00, 0.16},
	TuckBottomRight: {0.00, 0.16},
	TuckTopRight:    {0.16, 0.30},
	TuckBottomLeft:  {0.16, 0.30},
	PanelLeft:       {0.30, 0.52},
	PanelRight:      {0.30, 0.52},
	PanelBottom:     {0.52, 0.78},
	PanelTop:        {0.78, 1.00},
}

// Hinge is the line a panel rotates about, in world millimetres.
//
// A and B are the endpoints of the crease segment, taken from the flat
// sheet. Axis is the unit direction from A to B, and AngleDeg is the
// signed half-turn that folds the panel behind the front. The sign is
// chosen so that the panel moves towards -Z during the fold.
type Hinge struct {
	A        Vec3    `json:"a"`
	B        Vec3    `json:"b"`
	Axis     Vec3    `json:"axis"`
	AngleDeg float64 `json:"angle_deg"`
}

// Panel is one rigid piece of paper.
type Panel struct {
	ID string `json:"id"`
	// Parent is the panel this one is hinged to. It is empty for the
	// front panel, which is the root of the tree.
	Parent string `json:"parent"`
	// Role is a human readable label for the inspector.
	Role string `json:"role"`
	// Polygon is the panel in world coordinates, flat, z = 0. The
	// viewer builds one mesh from it and never rebuilds it.
	Polygon Polygon `json:"polygon"`
	// SheetPolygon is the same panel on the printed sheet, used by the
	// 2D overlay and by the PDF exporter.
	SheetPolygon Polygon `json:"sheet_polygon"`
	// UV is the texture coordinate of each polygon vertex, normalised
	// to the sheet and with v pointing up for the texture upload.
	UV Polygon `json:"uv"`
	// Hinge is nil for the front panel only.
	Hinge *Hinge `json:"hinge,omitempty"`
	// TStart and TEnd are the normalised fold progress window.
	TStart float64 `json:"t_start"`
	TEnd   float64 `json:"t_end"`
	// Layer orders the folded panels in depth. 0 is the front panel.
	Layer int `json:"layer"`
	// Tint is "seam" for the panels that show on the back of the
	// packet, so the viewer can shade them slightly.
	Tint string `json:"tint"`
}

// FrontPanel describes the visible face of the packet and the transform
// between the two frames.
//
// TransformFromFlat is the derivation the application is required to
// compute rather than hard-code: it maps a point on the flat sheet into
// the frame the address reads upright in. Its rotation is the diagonal
// of the printed Side B layout.
type FrontPanel struct {
	ID string `json:"id"`
	// Polygon is the panel on the flat sheet, in sheet coordinates.
	Polygon Polygon `json:"polygon"`
	// WorldPolygon is the same rectangle in world coordinates, upright
	// and centred on the origin.
	WorldPolygon Polygon `json:"world_polygon"`
	// TransformFromFlat maps sheet coordinates to panel coordinates.
	TransformFromFlat Transform `json:"transform_from_flat"`
	// TransformToFlat maps panel coordinates to sheet coordinates. It
	// is the transform the address and the stamp are printed with.
	TransformToFlat Transform `json:"transform_to_flat"`
	Size            Size      `json:"size"`
	RotationDeg     float64   `json:"rotation_deg"`
	Centre          Vec2      `json:"centre"`
}

// Crease is a fold line drawn on the flat sheet by the 2D overlay.
type Crease struct {
	ID string `json:"id"`
	// A and B are in sheet coordinates.
	A Vec2 `json:"a"`
	B Vec2 `json:"b"`
	// Kind is "edge" for the front panel's outline, "hinge" for an
	// edge flap, "tuck" for a diagonal corner fold and "hidden" for a
	// line the letter must not cross.
	Kind string `json:"kind"`
}

// Metrics are the computed properties of the fold.
type Metrics struct {
	PacketWidthMM  float64 `json:"packet_width_mm"`
	PacketHeightMM float64 `json:"packet_height_mm"`
	HeightIn       float64 `json:"height_in"`
	LengthIn       float64 `json:"length_in"`
	WithinLimits   bool    `json:"within_limits"`

	// FlapDepthMM is how deep each edge flap folds behind the panel.
	FlapDepthMM map[string]float64 `json:"flap_depth_mm"`

	// BackCoverage is the fraction of the front panel's back that at
	// least one flap covers at t = 1. A sealed packet is 1.
	BackCoverage float64 `json:"back_coverage"`
	// ProtrusionMM is how far the folded flaps reach beyond the packet
	// silhouette. The corner tucks leave a lip of at most TuckDepth.
	ProtrusionMM float64 `json:"protrusion_mm"`
	// SeamOverlapMM is the smallest overlap of a side flap over the
	// corner tucks.
	SeamOverlapMM float64 `json:"seam_overlap_mm"`
	// PartitionAreaMM2 and SheetAreaMM2 prove the panels tile the
	// sheet. They are equal when the geometry is sound.
	PartitionAreaMM2 float64 `json:"partition_area_mm2"`
	SheetAreaMM2     float64 `json:"sheet_area_mm2"`

	Warnings []string `json:"warnings"`
}

// Scene is everything the frontend and the exporter need to draw the
// fold. It is the single output of the engine.
type Scene struct {
	Params     Params     `json:"params"`
	Sheet      Size       `json:"sheet"`
	FrontPanel FrontPanel `json:"front_panel"`
	Panels     []Panel    `json:"panels"`
	FoldOrder  []string   `json:"fold_order"`
	Creases    []Crease   `json:"creases"`
	Layout     Layout     `json:"layout"`
	Metrics    Metrics    `json:"metrics"`

	// SheetPlacement maps panel coordinates onto the printed sheet. It
	// is the unmirrored placement used by Side B.
	SheetPlacement Placement `json:"sheet_placement"`
	// LetterPlacement maps the letter's own frame onto the sheet. It is
	// mirrored inside the panel because the letter reads from Side A.
	LetterPlacement Placement `json:"letter_placement"`
}

// Build computes the fold for the given parameters.
//
// It is deterministic and free of I/O: the same parameters always
// produce the same scene, which is what lets the frontend cache the
// scene and animate the hinges locally.
func Build(params Params) (Scene, error) {
	params = params.WithDefaults()
	if err := params.Validate(); err != nil {
		return Scene{}, err
	}

	frame := makeFrame(params)
	sheetPanel := sheetInPanelFrame(params, frame)
	front := Rect{X: 0, Y: 0, Width: params.PanelWidth, Height: params.PanelHeight}

	// The front panel must be inside the sheet; otherwise there is no
	// paper to fold. The rotation and the panel size decide this.
	if !insidePolygon(front.Polygon(), sheetPanel) {
		return Scene{}, fmt.Errorf("fold: the %0.1f x %0.1f mm panel at %0.1f degrees does not fit inside the %0.1f x %0.1f mm sheet",
			params.PanelWidth, params.PanelHeight, params.RotationDeg,
			params.SheetWidth, params.SheetHeight)
	}

	panels, err := partition(params, frame, sheetPanel)
	if err != nil {
		return Scene{}, err
	}

	scene := Scene{
		Params: params,
		Sheet:  Size{Width: params.SheetWidth, Height: params.SheetHeight},
		FrontPanel: FrontPanel{
			ID:                PanelFront,
			Polygon:           mapFrame(front.Polygon(), frame),
			WorldPolygon:      toWorld(front.Polygon(), params),
			TransformFromFlat: frameToTransform(frame).Inverse(),
			TransformToFlat:   frameToTransform(frame),
			Size:              params.Packet(),
			RotationDeg:       params.RotationDeg,
			Centre:            frame.PanelToSheet(Vec2{X: params.PanelWidth / 2, Y: params.PanelHeight / 2}),
		},
	}

	scene.Panels = panels
	scene.FoldOrder = foldOrder(panels)
	scene.Creases = creases(params, frame, panels)
	scene.Layout = makeLayout(params)
	scene.Metrics = measure(params, panels, sheetPanel)
	scene.SheetPlacement = scene.sheetPlacement()
	scene.LetterPlacement = scene.letterPlacement()

	return scene, nil
}

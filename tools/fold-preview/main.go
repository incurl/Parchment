// Command fold-preview renders the folded packet to PNG images.
//
// It is the headless half of the visual check: the same scene the 3D
// viewer consumes is folded at t = 1 and projected twice, from the front
// and from the back, so the packet's silhouette and the layer order can
// be checked without a display.
//
//	go run ./tools/fold-preview
//
// The images land in /tmp/parchment-fold-*.png.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"

	"github.com/incurl/parchment/internal/fold"
)

/** The output scale in pixels per millimetre. */
const scale = 4.0

/** The margin around the packet, in millimetres. */
const marginMM = 18.0

func main() {
	params := fold.DefaultParams()
	scene, err := fold.Build(params)
	if err != nil {
		fmt.Fprintln(os.Stderr, "build fold:", err)
		os.Exit(1)
	}

	widthMM := params.PanelWidth + 2*marginMM
	heightMM := params.PanelHeight + 2*marginMM

	front := newCanvas(widthMM, heightMM)
	back := newCanvas(widthMM, heightMM)

	// Sort by layer so each projection shows the real stacking: from the
	// front the front panel is nearest, from the back the deepest flap is.
	panels := append([]fold.Panel(nil), scene.Panels...)
	sort.SliceStable(panels, func(i, j int) bool { return panels[i].Layer > panels[j].Layer })

	for _, panel := range panels {
		polygon := scene.FoldedPolygon(panel, 1)
		front.fill(polygon, layerColour(panel.Layer), panel.ID == fold.PanelFront)
	}

	backPanels := append([]fold.Panel(nil), scene.Panels...)
	sort.SliceStable(backPanels, func(i, j int) bool { return backPanels[i].Layer < backPanels[j].Layer })
	for _, panel := range backPanels {
		back.fill(mirror(scene.FoldedPolygon(panel, 1)), layerColour(panel.Layer), panel.ID == fold.PanelFront)
	}

	// The printed content, on the front face only. The layout is in the
	// panel frame; the projection is in the world frame, where the panel
	// is centred on the origin with y up.
	toWorld := func(p fold.Vec2) fold.Vec2 {
		return fold.Vec2{X: p.X - params.PanelWidth/2, Y: params.PanelHeight/2 - p.Y}
	}
	toWorldPolygon := func(p fold.Polygon) fold.Polygon {
		out := make(fold.Polygon, len(p))
		for i, v := range p {
			out[i] = toWorld(v)
		}
		return out
	}
	front.stroke(toWorldPolygon(scene.Layout.AddressRect.Polygon()), colour{40, 40, 40, 255}, 1.5)
	front.stroke(toWorldPolygon(scene.Layout.ReturnRect.Polygon()), colour{140, 140, 140, 255}, 1)
	front.stroke(toWorldPolygon(scene.Layout.StampRect.Polygon()), colour{140, 140, 140, 255}, 1)

	write("/tmp/parchment-fold-front.png", front)
	write("/tmp/parchment-fold-back.png", back)

	m := scene.Metrics
	fmt.Printf("packet %.1f x %.1f mm (%.2f x %.2f in), %d panels\n",
		m.PacketWidthMM, m.PacketHeightMM, m.LengthIn, m.HeightIn, len(scene.Panels))
	fmt.Println("front: /tmp/parchment-fold-front.png, back: /tmp/parchment-fold-back.png")
}

/** colour is an RGBA pixel. */
type colour struct{ r, g, b, a uint8 }

/** layerColour gives each layer a distinguishable tint. */
func layerColour(layer int) colour {
	switch layer {
	case 0:
		return colour{252, 246, 232, 255} // the front panel
	case 1:
		return colour{226, 205, 168, 255} // corner tucks
	case 2:
		return colour{214, 186, 142, 255} // side flaps
	case 3:
		return colour{198, 166, 120, 255} // bottom flap
	default:
		return colour{178, 146, 102, 255} // final flap
	}
}

/** canvas is a millimetre-addressed image. */
type canvas struct {
	img  *image.RGBA
	w, h float64
}

/** newCanvas creates a white canvas of the given size in millimetres. */
func newCanvas(widthMM, heightMM float64) *canvas {
	w := int(math.Round(widthMM * scale))
	h := int(math.Round(heightMM * scale))
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	return &canvas{img: img, w: widthMM, h: heightMM}
}

/** project maps a world point to a pixel. */
func (c *canvas) project(p fold.Vec2) (float64, float64) {
	return (p.X + c.w/2) * scale, (c.h/2 - p.Y) * scale
}

/** fill paints a convex polygon. */
func (c *canvas) fill(polygon fold.Polygon, fill colour, outline bool) {
	if len(polygon) < 3 {
		return
	}

	// Even-odd scanline fill, which is exact for the convex panels.
	min, max := polygon.Bounds()
	y0 := int(math.Floor((c.h/2-max.Y)*scale)) - 1
	y1 := int(math.Ceil((c.h/2-min.Y)*scale)) + 1

	for py := maxInt(0, y0); py <= minInt(c.img.Bounds().Dy()-1, y1); py++ {
		wy := c.h/2 - (float64(py)+0.5)/scale
		var xs []float64
		for i := range polygon {
			a := polygon[i]
			b := polygon[(i+1)%len(polygon)]
			if (a.Y > wy) == (b.Y > wy) {
				continue
			}
			t := (wy - a.Y) / (b.Y - a.Y)
			xs = append(xs, a.X+(b.X-a.X)*t)
		}
		sort.Float64s(xs)
		for i := 0; i+1 < len(xs); i += 2 {
			px0 := int(math.Round((xs[i]+c.w/2)*scale - 0.5))
			px1 := int(math.Round((xs[i+1]+c.w/2)*scale - 0.5))
			for px := maxInt(0, px0); px <= minInt(c.img.Bounds().Dx()-1, px1); px++ {
				c.img.SetRGBA(px, py, color.RGBA{fill.r, fill.g, fill.b, fill.a})
			}
		}
	}

	if outline {
		c.stroke(polygon, colour{60, 52, 40, 255}, 1)
	}
}

/** stroke draws the polygon's outline. */
func (c *canvas) stroke(polygon fold.Polygon, ink colour, width float64) {
	for i := range polygon {
		c.line(polygon[i], polygon[(i+1)%len(polygon)], ink, width)
	}
}

/** line draws a straight segment with a simple thickness. */
func (c *canvas) line(a, b fold.Vec2, ink colour, width float64) {
	ax, ay := c.project(a)
	bx, by := c.project(b)
	steps := int(math.Max(math.Abs(bx-ax), math.Abs(by-ay))) + 1
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := ax + (bx-ax)*t
		y := ay + (by-ay)*t
		for dy := -width / 2; dy <= width/2; dy++ {
			for dx := -width / 2; dx <= width/2; dx++ {
				px, py := int(x+dx), int(y+dy)
				if px >= 0 && px < c.img.Bounds().Dx() && py >= 0 && py < c.img.Bounds().Dy() {
					c.img.SetRGBA(px, py, color.RGBA{ink.r, ink.g, ink.b, ink.a})
				}
			}
		}
	}
}

/** mirror flips a polygon horizontally, which is the back view. */
func mirror(polygon fold.Polygon) fold.Polygon {
	out := make(fold.Polygon, len(polygon))
	for i, p := range polygon {
		out[i] = fold.Vec2{X: -p.X, Y: p.Y}
	}
	return out
}

/** write saves the canvas as a PNG. */
func write(path string, c *canvas) {
	file, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer file.Close()
	if err := png.Encode(file, c.img); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

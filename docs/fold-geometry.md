# The fold geometry

This document explains the paper model and, in particular, the derivation
of the diagonal address transform. Every number the application prints
comes from this model; none of it is a constant in a renderer.

## 1. The frames

Two planar frames, both in millimetres:

- **The sheet frame**: `x` right, `y` down, origin at the sheet's
  top-left corner. This is the page. Side B is printed in this frame.
- **The panel frame**: `u` right, `v` down, origin at the front panel's
  top-left corner. This is the packet. The address reads upright here.

The front panel is a `W × H` rectangle in the panel frame, placed in the
sheet frame at an angle `φ` and an origin `O`:

```
p_sheet = O + R(φ) · p_panel
R(φ)    = [ cos φ  −sin φ ]
          [ sin φ   cos φ ]
```

`O` and `φ` are derived from the parameters, not written down:

```
φ = RotationDeg
centre_sheet = (sheet.W/2 + offsetX, sheet.H/2 + offsetY)
O = centre_sheet − R(φ) · (W/2, H/2)
```

Because `y` points down, a positive `φ` is clockwise on the printed page.

**`φ ≠ 0` is the whole point.** With `φ = 0` the front panel is parallel
to the sheet's edges and the address would print square. With `φ = 15°`
the panel is a rotated rectangle inside the sheet, and the address - laid
out upright in the panel frame - is printed diagonally.

## 2. The partition

The sheet is cut by the four lines through the panel's edges plus four
creases parallel to them, `TuckDepth = k` outside:

| Panel | Region (panel frame) | Hinge | Parent |
| --- | --- | --- | --- |
| `front` | `[0,W] × [0,H]` | none | - (root) |
| `left` | `u ≤ 0`, `−k ≤ v ≤ H+k` | `u = 0` | front |
| `right` | `u ≥ W`, `−k ≤ v ≤ H+k` | `u = W` | front |
| `top` | `0 ≤ u ≤ W`, `v ≤ 0` | `v = 0` | front |
| `bottom` | `0 ≤ u ≤ W`, `v ≥ H` | `v = H` | front |
| `tuck-tl` | `u ≤ 0`, `v ≤ −k` | `v = −k` | left |
| `tuck-tr` | `u ≥ W`, `v ≤ −k` | `v = −k` | right |
| `tuck-bl` | `u ≤ 0`, `v ≥ H+k` | `v = H+k` | left |
| `tuck-br` | `u ≥ W`, `v ≥ H+k` | `v = H+k` | right |

Each region is the sheet polygon clipped by half-planes, so the nine
polygons tile the sheet exactly. The engine asserts this: the sum of the
panel areas equals the sheet's area to within half a square millimetre,
and a sampled grid finds every sheet point in exactly one panel
(`TestPartitionTilesTheSheet`, `TestPartitionHasNoOverlaps`).

Why these cuts and not the storybook ones:

- The **corner tucks** exist because the sheet's corners lie outside the
  panel. A corner region touches two strips at once, so it cannot be
  hinged on a panel edge. Cutting it out at the corner crease gives it a
  single hinge - and gives the fold its diagonal creases, which are
  parallel to the panel's edges, hence diagonal on the sheet.
- The **`−k … H+k` band** on the side strips is what makes the tucks fold
  *inside* the packet rather than beside it. The strip keeps a `k`-wide
  margin at each end; the tuck is the rest of the corner, and folding it
  about `v = −k` lands it `k` to `H` away from the top edge, inside the
  panel.
- The cost is honest and measured: the side strips' `k`-wide margins fold
  behind the packet and stick out by `k`. That is the `overhang` metric,
  3 mm for the default fold, and it is what a real envelope's flap edges
  look like.

## 3. The hinge tree

Every panel has a parent and a hinge line on the shared edge. The front
panel is the root. This is a **kinematic chain**, not a mesh deformation:

```
M_front  = I
M_child  = M_parent · T(hinge.A) · R(hinge.axis, θ) · T(−hinge.A)
```

`θ` is the only thing that changes while folding:

```
θ_i(t) = angle_i · clamp((t − start_i) / (end_i − start_i), 0, 1)
```

with `angle_i = ±180°`, the sign chosen so the panel moves towards `−Z`
halfway through its fold. The schedule:

| t | Folds |
| --- | --- |
| 0.00–0.16 | the two opposite corner tucks (top-left, bottom-right) |
| 0.16–0.30 | the other two corner tucks |
| 0.30–0.52 | the left and right side flaps |
| 0.52–0.78 | the bottom flap |
| 0.78–1.00 | the top flap, sealing the seam |

**The invariant is structural.** Because the front panel is the root and
its matrix is the identity for every `t`, it is coplanar with the world
`XY` plane at all times, its normal is always `+Z`, and every other panel
is an affine image of the plane under a composition of half-turns - so
the folded packet is flat, and nothing but the front panel is at `z = 0`.
No crease can cross the front panel either: the cuts are outside it by
construction, and the test samples every hinge segment against an inset
copy of the panel rectangle (`TestFrontPanelIsInvariant`).

At `t = 1` the flaps have flipped their normal once (twice for a tuck,
which is carried by its parent flap), which is exactly the layer order
the renderer draws:

```
z = 0                 front panel, Side B up
z < 0, nearest        corner tucks
z < 0, then           left / right flaps
z < 0, then           bottom flap
z < 0, farthest       top flap (the visible back)
```

So from outside the packet every visible surface is Side B, and Side A is
inside.

## 4. The diagonal address transform

The address, the return address and the stamp are authored in the panel
frame: rectangles from `Layout`, with the text drawn upright along `+u`.

To print them on Side B, map through the panel-to-sheet transform:

```
p_sheet = TransformToFlat.Apply(p_panel)
        = R(φ) · p_panel + O
```

The **printed rotation** is therefore the direction of the panel's first
edge in the sheet frame:

```
φ_printed = atan2( (C1 − C0).y, (C1 − C0).x )
```

where `C0` and `C1` are two adjacent corners of the front panel polygon
as the partition produced it. `DerivedRotationDeg()` computes it that
way; the tests assert it equals the parameter to 1e-6 for a spread of
panel sizes, rotations and offsets, and that the printed address
rectangle's edges keep their lengths (a rotation, not a scale or shear).

`TransformFromFlat` is the inverse, and the round trip is exact:

```
TransformFromFlat.Apply(TransformToFlat.Apply(p)) == p      (1e-9)
```

That round trip is the acceptance test for this part of the geometry, and
it also runs in `tools/e2e-check` against a live build.

### Why the address is upright on the packet

The layout is expressed in the panel frame, and the packet *is* the panel
frame. Nothing else is involved. Changing the panel size, the rotation or
the offset changes `O`, `φ` and the panel polygon together, so the printed
position follows and the packet's copy stays upright. There is no stored
`45°` or `15°` in a renderer.

### The letter's frame (Side A)

Side A and Side B are opposite faces of the same paper: a design authored
in one appears mirrored from the other. The letter is authored in its own
left-to-right frame and mapped onto the panel with its horizontal axis
reversed:

```
u = W − a,   v = b
```

so it reads normally when the recipient opens the packet, and stays
inside the front panel. On the printed Side A page the application
mirrors the x coordinate back (`x_page = sheet.W − x_sheet`), which turns
the printed baseline into `−φ`; that is why the letter is very nearly
upright on the page while the address is at `+φ` on the other side. The
tests assert the mirror, the axis reversal and that the letter's safe
area stays inside the panel (`TestLetterPlacementReadsFromSideA`).

## 5. The metrics

Computed on every parameter change and shown in the UI:

- **Packet** - exactly `W × H`, because the front panel is one layer.
- **Back coverage** - a 1 mm grid over the panel's back, counting the
  points covered by at least one folded flap. The default fold is 100%.
- **Overhang** - the farthest a folded flap reaches past the packet's
  silhouette. The default is `k`, the corner-tuck lip.
- **Seam overlap** - the shallowest flap, the weakest seam.
- **Partition area** - the sum of the panel areas; a bug signal if it ever
  differs from the sheet's.
- **Postal limits** - the packet's shorter and longer side against the
  configured range (USPS by default: 3.5–6.125 in by 5–11.5 in).

Warnings are produced when the coverage is below 99.5%, the overhang
exceeds the tuck lip, the seam overlap is under the required minimum, the
packet is out of range, or the areas do not add up.

## 6. Checking it yourself

```bash
go test ./internal/fold/ -v      # the invariants, the transform, the metrics
go run ./tools/fold-preview      # /tmp/parchment-fold-{front,back}.png
```

The preview tool folds the scene at `t = 1` and projects it from both
sides. The front image is the acceptance criterion: a clean rectangle
with the address block, return address and stamp upright in it, and only
the `k`-wide tuck lips at the top and bottom edges. The back image shows
the flap stack and the seams.

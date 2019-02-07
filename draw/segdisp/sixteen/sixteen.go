// Copyright 2019 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package sixteen simulates a 16-segment display drawn on a canvas.

Given a canvas, determines the placement and size of the individual
segments and exposes API that can turn individual segments on and off or
display ASCII characters.

The following outlines segments in the display and their names.

       A1      A2
     ------- -------
    | \     |     / |
    |  \    |    /  |
  F |   H   J   K   | B
    |    \  |  /    |
    |     \ | /     |
     -G1---- ----G2-
    |     / | \     |
    |    /  |  \    |
  E |   N   M   L   | C
    |  /    |    \  |
    | /     |     \ |
     ------- -------
       D1      D2
*/
package sixteen

import (
	"bytes"
	"fmt"
	"image"
	"math"

	"github.com/mum4k/termdash/area"
	"github.com/mum4k/termdash/canvas"
	"github.com/mum4k/termdash/canvas/braille"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/draw/segdisp/segment"
)

// Segment represents a single segment in the display.
type Segment int

// String implements fmt.Stringer()
func (s Segment) String() string {
	if n, ok := segmentNames[s]; ok {
		return n
	}
	return "SegmentUnknown"
}

// segmentNames maps Segment values to human readable names.
var segmentNames = map[Segment]string{
	A1: "A1",
	A2: "A2",
	B:  "B",
	C:  "C",
	D1: "D1",
	D2: "D2",
	E:  "E",
	F:  "F",
	G1: "G1",
	G2: "G2",
	H:  "H",
	J:  "J",
	K:  "K",
	L:  "L",
	M:  "M",
	N:  "N",
}

const (
	segmentUnknown Segment = iota

	A1
	A2
	B
	C
	D1
	D2
	E
	F
	G1
	G2
	H
	J
	K
	L
	M
	N

	segmentMax // Used for validation.
)

// characterSegments maps characters that can be displayed on their segments.
// See doc/16-Segment-ASCII-All.jpg and:
// https://www.partsnotincluded.com/electronics/segmented-led-display-ascii-library
var characterSegments = map[rune][]Segment{
	' ':  nil,
	'!':  {B, C},
	'"':  {J, B},
	'#':  {J, B, G1, G2, M, C, D1, D2},
	'$':  {A1, A2, F, J, G1, G2, M, C, D1, D2},
	'%':  {A1, F, J, K, G1, G2, N, M, C, D2},
	'&':  {A1, H, J, G1, E, L, D1, D2},
	'\'': {J},
	'(':  {K, L},
	')':  {H, N},
	'*':  {H, J, K, G1, G2, N, M, L},
	'+':  {J, G1, G2, M},
	',':  {N},
	'-':  {G1, G2},
	'/':  {N, K},

	'0': {A1, A2, F, K, B, E, N, C, D1, D2},
	'1': {K, B, C},
	'2': {A1, A2, B, G1, G2, E, D1, D2},
	'3': {A1, A2, B, G2, C, D1, D2},
	'4': {F, B, G1, G2, C},
	'5': {A1, A2, F, G1, L, D1, D2},
	'6': {A1, A2, F, G1, G2, E, C, D1, D2},
	'7': {A1, A2, B, C},
	'8': {A1, A2, F, B, G1, G2, E, C, D1, D2},
	'9': {A1, A2, F, B, G1, G2, C, D1, D2},

	':': {J, M},
	';': {J, N},
	'<': {K, G1, L},
	'=': {G1, G2, D1, D2},
	'>': {H, G2, N},
	'?': {A1, A2, B, G2, M},
	'@': {A1, A2, F, J, B, G2, E, D1, D2},

	'A': {A1, A2, F, B, G1, G2, E, C},
	'B': {A1, A2, J, B, G2, M, C, D1, D2},
	'C': {A1, A2, F, E, D1, D2},
	'D': {A1, A2, J, B, M, C, D1, D2},
	'E': {A1, A2, F, G1, E, D1, D2},
	'F': {A1, A2, F, G1, E},
	'G': {A1, A2, F, G2, E, C, D1, D2},
	'H': {F, B, G1, G2, E, C},
	'I': {A1, A2, J, M, D1, D2},
	'J': {B, E, C, D1, D2},
	'K': {F, K, G1, E, L},
	'L': {F, E, D1, D2},
	'M': {F, H, K, B, E, C},
	'N': {F, H, B, E, L, C},
	'O': {A1, A2, F, B, E, C, D1, D2},
	'P': {A1, A2, F, B, G1, G2, E},
	'Q': {A1, A2, F, B, E, L, C, D1, D2},
	'R': {A1, A2, F, B, G1, G2, E, L},
	'S': {A1, A2, F, G1, G2, C, D1, D2},
	'T': {A1, A2, J, M},
	'U': {F, B, E, C, D1, D2},
	'V': {F, K, E, N},
	'W': {F, E, N, L, C, B},
	'X': {H, K, N, L},
	'Y': {F, B, G1, G2, C, D1, D2},
	'Z': {A1, A2, K, N, D1, D2},

	'[':  {A2, J, M, D2},
	'\\': {H, L},
	']':  {A1, J, M, D1},
	'^':  {N, L},
	'_':  {D1, D2},
	'`':  {H},

	'a': {G1, E, M, D1, D2},
	'b': {F, G1, E, M, D1},
	'c': {G1, E, D1},
	'd': {B, G2, M, C, D2},
	'e': {G1, E, N, D1},
	'f': {A2, J, G1, G2, M},
	'g': {A1, F, J, G1, M, D1},
	'h': {F, G1, E, M},
	'i': {M},
	'j': {J, E, M, D1},
	'k': {J, K, M, L},
	'l': {F, E},
	'm': {G1, G2, E, M, C},
	'n': {G1, E, M},
	'o': {G1, E, M, D1},
	'p': {A1, F, J, G1, E},
	'q': {A1, F, J, G1, M},
	'r': {G1, E},
	's': {A1, F, G1, M, D1},
	't': {F, G1, E, D1},
	'u': {E, M, D1},
	'v': {E, N},
	'w': {E, N, L, C},
	'x': {H, K, N, L},
	'y': {J, B, G2, C, D2},
	'z': {G1, N, D1},

	'{': {A2, J, G1, M, D2},
	'|': {J, M},
	'}': {A1, J, G2, M, D1},
	'~': {K, G1, G2, N},
}

// SupportsChars asserts whether the display supports all runes in the
// provided string.
// The display only supports a subset of ASCII characters.
// Returns any unsupported runes found in the string in an unspecified order.
func SupportsChars(s string) (bool, []rune) {
	unsupp := map[rune]bool{}
	for _, r := range s {
		if _, ok := characterSegments[r]; !ok {
			unsupp[r] = true
		}
	}

	var res []rune
	for r := range unsupp {
		res = append(res, r)
	}
	return len(res) == 0, res
}

// Sanitize returns a copy of the string, replacing all unsupported characters
// with a space character.
func Sanitize(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		if _, ok := characterSegments[r]; !ok {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// AllSegments returns all 16 segments in an undefined order.
func AllSegments() []Segment {
	var res []Segment
	for s := range segmentNames {
		res = append(res, s)
	}
	return res
}

// Option is used to provide options.
type Option interface {
	// set sets the provided option.
	set(*Display)
}

// option implements Option.
type option func(*Display)

// set implements Option.set.
func (o option) set(d *Display) {
	o(d)
}

// CellOpts sets the cell options on the cells that contain the segment display.
func CellOpts(cOpts ...cell.Option) Option {
	return option(func(d *Display) {
		d.cellOpts = cOpts
	})
}

// Display represents the segment display.
// This object is not thread-safe.
type Display struct {
	// segments maps segments to their current status.
	segments map[Segment]bool

	cellOpts []cell.Option
}

// New creates a new segment display.
// Initially all the segments are off.
func New(opts ...Option) *Display {
	d := &Display{
		segments: map[Segment]bool{},
	}

	for _, opt := range opts {
		opt.set(d)
	}
	return d
}

// Clear clears the entire display, turning all segments off.
func (d *Display) Clear(opts ...Option) {
	for _, opt := range opts {
		opt.set(d)
	}

	d.segments = map[Segment]bool{}
}

// SetSegment sets the specified segment on.
// This method is idempotent.
func (d *Display) SetSegment(s Segment) error {
	if s <= segmentUnknown || s >= segmentMax {
		return fmt.Errorf("unknown segment %v(%d)", s, s)
	}
	d.segments[s] = true
	return nil
}

// ClearSegment sets the specified segment off.
// This method is idempotent.
func (d *Display) ClearSegment(s Segment) error {
	if s <= segmentUnknown || s >= segmentMax {
		return fmt.Errorf("unknown segment %v(%d)", s, s)
	}
	d.segments[s] = false
	return nil
}

// ToggleSegment toggles the state of the specified segment, i.e it either sets
// or clears it depending on its current state.
func (d *Display) ToggleSegment(s Segment) error {
	if s <= segmentUnknown || s >= segmentMax {
		return fmt.Errorf("unknown segment %v(%d)", s, s)
	}
	if d.segments[s] {
		d.segments[s] = false
	} else {
		d.segments[s] = true
	}
	return nil
}

// Character sets all the segments that are needed to display the provided character.
// The display only supports a subset of ASCII characters, use SupportsChars()
// or Sanitize() to ensure the provided character is supported.
// Doesn't clear the display of segments set previously.
func (d *Display) SetCharacter(c rune) error {
	seg, ok := characterSegments[c]
	if !ok {
		return fmt.Errorf("display doesn't support character %q rune(%v)", c, c)
	}

	for _, s := range seg {
		if err := d.SetSegment(s); err != nil {
			return err
		}
	}
	return nil
}

// Minimum valid size of a cell canvas in order to draw the segment display.
const (
	// MinCols is the smallest valid amount of columns in a cell area.
	MinCols = 6
	// MinRowPixels is the smallest valid amount of rows in a cell area.
	MinRows = 5
)

// aspectRatio is the desired aspect ratio of a single segment display.
var aspectRatio = image.Point{3, 5}

// Draw draws the current state of the segment display onto the canvas.
// The canvas must be at least MinCols x MinRows cells, or an error will be
// returned.
// Any options provided to draw overwrite the values provided to New.
func (d *Display) Draw(cvs *canvas.Canvas, opts ...Option) error {
	for _, o := range opts {
		o.set(d)
	}

	bc, bcAr, err := toBraille(cvs)
	if err != nil {
		return err
	}

	attr := newAttributes(bcAr)
	var sOpts []segment.Option
	if len(d.cellOpts) > 0 {
		sOpts = append(sOpts, segment.CellOpts(d.cellOpts...))
	}
	for _, segArg := range []struct {
		s    Segment
		opts []segment.Option
	}{
		{A1, nil},
		{A2, nil},

		{F, nil},
		{J, []segment.Option{segment.SkipSlopesLTE(2)}},
		{B, []segment.Option{segment.ReverseSlopes()}},

		{G1, []segment.Option{segment.SkipSlopesLTE(2)}},
		{G2, []segment.Option{segment.SkipSlopesLTE(2)}},

		{E, nil},
		{M, []segment.Option{segment.SkipSlopesLTE(2)}},
		{C, []segment.Option{segment.ReverseSlopes()}},

		{D1, []segment.Option{segment.ReverseSlopes()}},
		{D2, []segment.Option{segment.ReverseSlopes()}},
	} {
		if !d.segments[segArg.s] {
			continue
		}
		sOpts := append(sOpts, segArg.opts...)
		ar := attr.hvSegArea(segArg.s)
		if err := segment.HV(bc, ar, hvSegType[segArg.s], sOpts...); err != nil {
			return fmt.Errorf("failed to draw segment %v, segment.HV => %v", segArg.s, err)
		}
	}

	var dsOpts []segment.DiagonalOption
	if len(d.cellOpts) > 0 {
		dsOpts = append(dsOpts, segment.DiagonalCellOpts(d.cellOpts...))
	}
	for _, seg := range []Segment{H, K, N, L} {
		if !d.segments[seg] {
			continue
		}
		ar := attr.diaSegArea(seg)
		if err := segment.Diagonal(bc, ar, attr.segSize, diaSegType[seg], dsOpts...); err != nil {
			return fmt.Errorf("failed to draw segment %v, segment.Diagonal => %v", seg, err)
		}
	}
	return bc.CopyTo(cvs)
}

// Required, when given an area of cells, returns either an area of the same
// size or a smaller area that is required to draw one display.
// Returns a smaller area when the provided area didn't have the required
// aspect ratio.
// Returns an error if the area is too small to draw a segment display, i.e.
// smaller than MinCols x MinRows.
func Required(cellArea image.Rectangle) (image.Rectangle, error) {
	if cols, rows := cellArea.Dx(), cellArea.Dy(); cols < MinCols || rows < MinRows {
		return image.ZR, fmt.Errorf("cell area %v is too small to draw the segment display, has %dx%d cells, need at least %dx%d cells",
			cellArea, cols, rows, MinCols, MinRows)
	}

	bcAr := image.Rect(cellArea.Min.X, cellArea.Min.Y, cellArea.Max.X*braille.ColMult, cellArea.Max.Y*braille.RowMult)
	bcArAdj := area.WithRatio(bcAr, aspectRatio)

	needCols := int(math.Ceil(float64(bcArAdj.Dx()) / braille.ColMult))
	needRows := int(math.Ceil(float64(bcArAdj.Dy()) / braille.RowMult))
	needAr := image.Rect(cellArea.Min.X, cellArea.Min.Y, cellArea.Min.X+needCols, cellArea.Min.Y+needRows)
	return needAr, nil
}

// toBraille converts the canvas into a braille canvas and returns a pixel area
// with aspect ratio adjusted for the segment display.
func toBraille(cvs *canvas.Canvas) (*braille.Canvas, image.Rectangle, error) {
	ar, err := Required(cvs.Area())
	if err != nil {
		return nil, image.ZR, fmt.Errorf("Required => %v", err)
	}

	bc, err := braille.New(ar)
	if err != nil {
		return nil, image.ZR, fmt.Errorf("braille.New => %v", err)
	}
	return bc, area.WithRatio(bc.Area(), aspectRatio), nil
}
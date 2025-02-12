// Copyright 2024 The TCell Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tcell

import (
	"os"
	"reflect"

	runewidth "github.com/mattn/go-runewidth"
)

type cell struct {
	currMain  rune
	currComb  []rune
	currStyle Style
	lastMain  rune
	lastStyle Style
	lastComb  []rune
	width     int
	lock      bool
}

// CellBuffer represents a two-dimensional array of character cells.
// This is primarily intended for use by Screen implementors; it
// contains much of the common code they need.  To create one, just
// declare a variable of its type; no explicit initialization is necessary.
//
// CellBuffer is not thread safe.
type CellBuffer struct {
	w     int
	h     int
	cells []cell
}

// SetContent sets the contents (primary rune, combining runes,
// and style) for a cell at a given location.  If the background or
// foreground of the style is set to ColorNone, then the respective
// color is left un changed.
func (cb *CellBuffer) SetContent(x int, y int,
	mainc rune, combc []rune, style Style,
) {
	if x >= 0 && y >= 0 && x < cb.w && y < cb.h {
		c := &cb.cells[(y*cb.w)+x]

		// Wide characters: we want to mark the "wide" cells
		// dirty as well as the base cell, to make sure we consider
		// both cells as dirty together.  We only need to do this
		// if we're changing content
		if (c.width > 0) && (mainc != c.currMain || len(combc) != len(c.currComb) || (len(combc) > 0 && !reflect.DeepEqual(combc, c.currComb))) {
			for i := 0; i < c.width; i++ {
				cb.SetDirty(x+i, y, true)
			}
		}

		c.currComb = append([]rune{}, combc...)

		if c.currMain != mainc {
			c.width = runewidth.RuneWidth(mainc)
		}
		c.currMain = mainc
		if style.fg == ColorNone {
			style.fg = c.currStyle.fg
		}
		if style.bg == ColorNone {
			style.bg = c.currStyle.bg
		}
		c.currStyle = style
	}
}

// GetContent returns the contents of a character cell, including the
// primary rune, any combining character runes (which will usually be
// nil), the style, and the display width in cells.  (The width can be
// either 1, normally, or 2 for East Asian full-width characters.)
func (cb *CellBuffer) GetContent(x, y int) (rune, []rune, Style, int) {
	var mainc rune
	var combc []rune
	var style Style
	var width int
	if x >= 0 && y >= 0 && x < cb.w && y < cb.h {
		c := &cb.cells[(y*cb.w)+x]
		mainc, combc, style = c.currMain, c.currComb, c.currStyle
		if width = c.width; width == 0 || mainc < ' ' {
			width = 1
			mainc = ' '
		}
	}
	return mainc, combc, style, width
}

// Size returns the (width, height) in cells of the buffer.
func (cb *CellBuffer) Size() (int, int) {
	return cb.w, cb.h
}

// Invalidate marks all characters within the buffer as dirty.
func (cb *CellBuffer) Invalidate() {
	for i := range cb.cells {
		cb.cells[i].lastMain = rune(0)
	}
}

// Dirty checks if a character at the given location needs to be
// refreshed on the physical display.  This returns true if the cell
// content is different since the last time it was marked clean.
func (cb *CellBuffer) Dirty(x, y int) bool {
	if x >= 0 && y >= 0 && x < cb.w && y < cb.h {
		c := &cb.cells[(y*cb.w)+x]
		if c.lock {
			return false
		}
		if c.lastMain == rune(0) {
			return true
		}
		if c.lastMain != c.currMain {
			return true
		}
		if c.lastStyle != c.currStyle {
			return true
		}
		if len(c.lastComb) != len(c.currComb) {
			return true
		}
		for i := range c.lastComb {
			if c.lastComb[i] != c.currComb[i] {
				return true
			}
		}
	}
	return false
}

// SetDirty is normally used to indicate that a cell has
// been displayed (in which case dirty is false), or to manually
// force a cell to be marked dirty.
func (cb *CellBuffer) SetDirty(x, y int, dirty bool) {
	if x >= 0 && y >= 0 && x < cb.w && y < cb.h {
		c := &cb.cells[(y*cb.w)+x]
		if dirty {
			c.lastMain = rune(0)
		} else {
			if c.currMain == rune(0) {
				c.currMain = ' '
			}
			c.lastMain = c.currMain
			c.lastComb = c.currComb
			c.lastStyle = c.currStyle
		}
	}
}

// LockCell locks a cell from being drawn, effectively marking it "clean" until
// the lock is removed. This can be used to prevent tcell from drawing a given
// cell, even if the underlying content has changed. For example, when drawing a
// sixel graphic directly to a TTY screen an implementer must lock the region
// underneath the graphic to prevent tcell from drawing on top of the graphic.
func (cb *CellBuffer) LockCell(x, y int) {
	if x < 0 || y < 0 {
		return
	}
	if x >= cb.w || y >= cb.h {
		return
	}
	c := &cb.cells[(y*cb.w)+x]
	c.lock = true
}

// UnlockCell removes a lock from the cell and marks it as dirty
func (cb *CellBuffer) UnlockCell(x, y int) {
	if x < 0 || y < 0 {
		return
	}
	if x >= cb.w || y >= cb.h {
		return
	}
	c := &cb.cells[(y*cb.w)+x]
	c.lock = false
	cb.SetDirty(x, y, true)
}

// Resize is used to resize the cells array, with different dimensions,
// while preserving the original contents.  The cells will be invalidated
// so that they can be redrawn.
func (cb *CellBuffer) Resize(w, h int) {
	if cb.h == h && cb.w == w {
		return
	}

	newc := make([]cell, w*h)
	for y := 0; y < h && y < cb.h; y++ {
		for x := 0; x < w && x < cb.w; x++ {
			oc := &cb.cells[(y*cb.w)+x]
			nc := &newc[(y*w)+x]
			nc.currMain = oc.currMain
			nc.currComb = oc.currComb
			nc.currStyle = oc.currStyle
			nc.width = oc.width
			nc.lastMain = rune(0)
		}
	}
	cb.cells = newc
	cb.h = h
	cb.w = w
}

// Fill fills the entire cell buffer array with the specified character
// and style.  Normally choose ' ' to clear the screen.  This API doesn't
// support combining characters, or characters with a width larger than one.
// If either the foreground or background are ColorNone, then the respective
// color is unchanged.
func (cb *CellBuffer) Fill(r rune, style Style) {
	for i := range cb.cells {
		c := &cb.cells[i]
		c.currMain = r
		c.currComb = nil
		cs := style
		if cs.fg == ColorNone {
			cs.fg = c.currStyle.fg
		}
		if cs.bg == ColorNone {
			cs.bg = c.currStyle.bg
		}
		c.currStyle = cs
		c.width = 1
	}
}

var runeConfig *runewidth.Condition

func init() {
	// The defaults for the runewidth package are poorly chosen for terminal
	// applications.  We however will honor the setting in the environment if
	// it is set.
	if os.Getenv("RUNEWIDTH_EASTASIAN") == "" {
		runewidth.DefaultCondition.EastAsianWidth = false
	}
}

// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/hailiang/color"
	"go-mupdf/fz"
	//	"sync"
	//	"time"
	"unsafe"
)

type RenderOption struct {
	Width      int
	Dark       bool
	Brightness float32
}

type Model struct {
	ctx *fz.Context
	doc *fz.Document

	// render cache
	RenderOption
	height  int
	heights []int
	cache   []Pixel
	cached  []bool
	stop    bool
	//	mu      sync.Mutex // cache lock
	//	wg      sync.WaitGroup
}

func newModel() (*Model, error) {
	ctx := fz.AllocDefault.NewContext(nil, fz.StoreDefault)
	return &Model{
		ctx: ctx,
		RenderOption: RenderOption{
			Dark:       true,
			Brightness: 0.5,
		},
	}, nil
}

func (m *Model) Open(file string) error {
	if m.doc != nil {
		m.doc.Close()
		m.doc = nil
	}
	m.doc = m.ctx.OpenDocument(file)

	return nil
}

func (m *Model) PageCount() int {
	return m.doc.CountPages()
}

func (m *Model) Height() int {
	return len(m.cache) / m.RenderOption.Width
}

func (m *Model) setWidth(width int) {
	if width != m.Width {
		m.Width = width

		m.heights = make([]int, m.PageCount())
		m.height = 0
		for i := 0; i < m.PageCount(); i++ {
			page := m.doc.LoadPage(i)
			var bounds fz.Rect
			m.doc.BoundPage(page, &bounds)
			zoom := float32(width) / bounds.X1
			var transform fz.Matrix
			transform.Scale(zoom, zoom)
			bounds.Transform(&transform)
			var bbox fz.Irect
			bbox.RoundRect(&bounds)
			m.height += int(bbox.Y1)
			m.heights[i] = int(bbox.Y1)
			m.doc.FreePage(page)
		}
		m.cache = make([]Pixel, m.Width*m.height)
		m.cached = make([]bool, m.PageCount())
	}
}

func (m *Model) FlipDarkMode() {
	m.Dark = !m.Dark
	m.cached = make([]bool, m.PageCount())
}

/*
func (m *Model) stopEagerLoad() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stop = true
	m.wg.Wait()
	m.stop = false
}

func (m *Model) eagerLoad() {
	m.stopEagerLoad()
	m.wg.Add(1)
	go func() {
		for i := 0; i < m.PageCount(); i++ {
			if m.stop {
				break
			}
			if m.cached[i] {
				continue
			}
			m.renderPage(i)
			time.Sleep(10 * time.Millisecond)
		}
		m.wg.Done()
	}()
}
*/

func (m *Model) renderPage(pageIndex int) {
	if m.cached[pageIndex] {
		return
	}
	pageStart := 0
	for i := 0; i < pageIndex; i++ {
		pageStart += m.heights[i]
	}
	renderPage(m.ctx, m.doc, pageIndex, &m.RenderOption, m.cache[m.Width*pageStart:])
	m.cached[pageIndex] = true
}

func (m *Model) Get(y, h, w int) []Pixel {
	m.setWidth(w)
	pageStart := 0
	for i := 0; i < m.PageCount(); i++ {
		pageEnd := pageStart + m.heights[i]
		if !m.cached[i] {
			if pageEnd >= y && pageStart <= y+h {
				m.renderPage(i)
			}
		}
		pageStart = pageEnd
	}

	srcStart := m.Width * y
	srcEnd := srcStart + m.Width*h
	if srcEnd > len(m.cache) {
		srcEnd = len(m.cache)
	}
	return m.cache[srcStart:srcEnd]

}

func renderPage(ctx *fz.Context, doc *fz.Document, pageIndex int, opt *RenderOption, buf []Pixel) {
	page := doc.LoadPage(pageIndex)
	defer doc.FreePage(page)

	var bounds fz.Rect
	doc.BoundPage(page, &bounds)
	zoom := float32(opt.Width) / bounds.X1
	var transform fz.Matrix
	transform.Scale(zoom, zoom)
	bounds.Transform(&transform)
	var bbox fz.Irect
	bbox.RoundRect(&bounds)

	pix := ctx.NewPixmapWithBboxAndData(ctx.DeviceRgb(), &bbox,
		(*byte)(unsafe.Pointer(&buf[0])))
	ctx.ClearPixmapWithValue(pix, 0xFF)

	dev := fz.NewDeviceHook(ctx.NewDrawDevice(pix))
	defer dev.Free()

	var imgRects Rects
	dev.ImageHook = func(image *fz.Image, ctm *fz.Matrix, alpha float32) {
		pt := fz.Point{0, 0}
		pt.Transform(ctm)
		x, y := int(pt.X), int(pt.Y)
		w := round(sqrtf(ctm.A*ctm.A+ctm.B*ctm.B) + 1)
		h := round(sqrtf(ctm.C*ctm.C+ctm.D*ctm.D) + 1)
		imgRects = append(imgRects, &Rect{x, y, w, h})
	}

	doc.RunPage(page, dev.ToDevice(), &transform, nil)

	if opt.Dark {
		pixels := buf
		off := 0
		l := opt.Brightness
		ll := opt.Brightness * opt.Brightness

		// shortcut for white
		rgbWhite := Pixel{0xFF, 0xFF, 0xFF, 0xFF}
		hsvWhite := color.RGBToHSVi(0xFF, 0xFF, 0xFF)
		hsvIn, hsvOut := hsvWhite, hsvWhite
		hsvOut.V = int16(l * float32(color.Fac-hsvOut.V))
		hsvIn.V = int16(ll * float32(hsvIn.V))

		for y := 0; y < int(pix.H); y++ {
			for x := 0; x < int(pix.W); x++ {
				pixel := pixels[off+x]
				if pixel == rgbWhite {
					if imgRects.In(x, y) {
						pixels[off+x].SetRGB(hsvIn.ToRGB())
					} else {
						pixels[off+x].SetRGB(hsvOut.ToRGB())
					}
				} else {
					hsv := color.RGBToHSVi(pixels[off+x].GetRGB())
					if !imgRects.In(x, y) {
						hsv.V = int16(l * float32(color.Fac-hsv.V))
					} else {
						hsv.V = int16(ll * float32(hsv.V))
					}
					pixels[off+x].SetRGB(hsv.ToRGB())
				}
			}
			off += int(pix.W)
		}
	}
}

func (m *Model) disposePageCache() {
	m.cached = nil
	m.cache = nil
}

func (m *Model) Dispose() {
	m.disposePageCache()
	if m.doc != nil {
		m.doc.Close()
		m.doc = nil
	}
	if m.ctx != nil {
		m.ctx.FreeContext()
		m.ctx = nil
	}
}

type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

func (r *Rect) In(x, y int) bool {
	return x >= r.X && x < r.X+r.Width &&
		y >= r.Y && y < r.Y+r.Height
}

type Rects []*Rect

func (rs Rects) In(x, y int) bool {
	for _, r := range rs {
		if r.In(x, y) {
			return true
		}
	}
	return false
}

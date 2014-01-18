// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"go-sdl"
	//	"go-sdl/ttf"
	"time"
	"unsafe"
)

const (
	Title         = "gr"
	DefaultWidth  = 640
	DefaultHeight = 480
	FPS           = 60
)

type View struct {
	win *sdl.Window
	ren *sdl.Renderer
	tex *Texture
}

func newView(sur Surface) (*View, error) {
	win := sdl.CreateWindow(Title, sdl.WindowposUndefined,
		sdl.WindowposUndefined, DefaultWidth, DefaultHeight, sdl.WindowShown)
	if win == nil {
		return nil, sdlError()
	}
	ren := win.CreateRenderer(-1, sdl.RendererAccelerated|sdl.RendererPresentvsync)
	if ren == nil {
		return nil, sdlError()
	}
	w, h := win.GetSize()
	tex, err := NewTexture(w, h, 0, ren, sur)
	if err != nil {
		return nil, err
	}
	view := &View{win: win, ren: ren, tex: tex}
	return view, nil
}

func (v *View) Height() int {
	_, h := v.win.GetSize()
	return h
}

func (v *View) Width() int {
	w, _ := v.win.GetSize()
	return w
}

func (v *View) Reload() error {
	oldW, oldY := v.tex.Width(), v.tex.y
	newW := v.Width()
	newY := round(float32(oldY) * float32(newW) / float32(oldW))
	v.tex.Dispose()
	tex, err := NewTexture(v.Width(), v.Height(), newY, v.ren, v.tex.sur)
	if err != nil {
		return err
	}
	tex.Init()
	v.tex = tex
	return nil
}

func (v *View) Dispose() {
	if v.tex != nil {
		v.tex.Dispose()
		v.tex = nil
	}
	if v.ren != nil {
		v.ren.Destroy()
		v.ren = nil
	}
	if v.win != nil {
		v.win.Destroy()
		v.win = nil
	}
}

func (v *View) Render() error {
	if v.tex != nil {
		if err := v.tex.Render(v.ren); err != nil {
			return err
		}
		v.ren.RenderPresent()
	}
	return nil
}

func (v *View) SmoothScroll(offset int, duration float64) error {
	if !v.tex.CanScroll(offset) {
		return nil
	}
	t := time.Now()
	moved := 0
	speed := float64(offset) / duration
	for {
		du := time.Since(t)
		if du.Seconds() < 1/FPS {
			continue
		}
		inc := int(speed*du.Seconds() + 0.5)
		if diff := offset - moved; abs(inc) > abs(diff) {
			inc = diff
		}
		if inc != 0 {
			if err := v.tex.Scroll(inc); err != nil {
				return err
			}
			if err := v.Render(); err != nil {
				return err
			}
			moved += inc
			t = t.Add(du)
		}
		if moved == offset {
			break
		}
	}
	return nil
}

type Surface interface {
	Height() int
	Get(y, h, w int) []Pixel
}

// Texture that supports wrap around.
type Texture struct {
	tex  *sdl.Texture
	zero int
	sur  Surface
	y    int
}

func NewTexture(w, h, y int, ren *sdl.Renderer, sur Surface) (*Texture, error) {
	pixelFormat := sdl.PixelformatAbgr8888
	tex := ren.CreateTexture(pixelFormat, sdl.TextureaccessStreaming, w, h)
	if tex == nil {
		return nil, sdlError()
	}
	t := &Texture{tex, 0, sur, y}
	return t, nil
}

func (t *Texture) Init() error {
	y := t.y
	th := t.Height()
	pix := t.sur.Get(y, th, t.Width())
	if err := t.copyFrom(t.zero, pix); err != nil {
		return err
	}
	return nil
}

func (t *Texture) CanScroll(offset int) bool {
	if offset < 0 {
		return t.y > 0
	}
	bottom := t.sur.Height() - t.Height()
	return t.y < bottom
}

func (t *Texture) Scroll(offset int) error {
	if offset > 0 {
		return t.scrollDown(offset)
	} else if offset < 0 {
		return t.scrollUp(-offset)
	}
	return nil
}

func (t *Texture) scrollDown(h int) error {
	y := t.y
	th := t.Height()
	bottom := t.sur.Height() - th
	if y < bottom {
		if y+h > bottom {
			h = bottom - y
		}
		if mul := h / th; mul >= 1 {
			y += mul * th
			h -= mul * th
			if h == 0 {
				h = th
			}
		}

		pix := t.sur.Get(y+th, h, t.Width())
		ih := int(len(pix)) / t.Width()
		if err := t.copyFrom(t.zero, pix); err != nil {
			return err
		}
		t.moveZero(ih)
		y += h
		t.y = y
	}
	return nil
}

func (t *Texture) scrollUp(h int) error {
	y := t.y
	th := t.Height()
	if y > 0 {
		if y-h < 0 {
			h = y
		}
		if mul := h / th; mul >= 1 {
			y -= mul * th
			h -= mul * th
			if h == 0 {
				h = th
			}
		}
		pix := t.sur.Get(y-h, h, t.Width())
		ih := int(len(pix)) / t.Width()
		t.moveZero(-ih)
		if err := t.copyFrom(t.zero, pix); err != nil {
			return err
		}
		y -= h
		t.y = y
	}
	return nil
}

func (t *Texture) copyFrom(zero_ int, pixels []Pixel) error {
	if len(pixels) == 0 {
		return nil
	}
	zero := int32(zero_)
	w := int32(t.Width())
	h := int32(len(pixels)) / w
	th := int32(t.Height())
	h1 := h
	if h > th-zero {
		h1 = th - zero
	}
	rt := &sdl.Rect{
		Y: zero,
		W: w,
		H: h1,
	}
	if t.tex.Update(rt, uintptr(unsafe.Pointer(&pixels[0])), t.Pitch()) != 0 {
		return sdlError()
	}
	if h2 := h - h1; h2 > 0 {
		rt := &sdl.Rect{
			Y: 0,
			W: w,
			H: h2,
		}
		if t.tex.Update(rt, uintptr(unsafe.Pointer(&pixels[int(w*h1)])), t.Pitch()) != 0 {
			return sdlError()
		}
	}
	return nil
}

func (t *Texture) moveZero(offset int) {
	t.zero += offset
	th := t.Height()
	for t.zero < 0 {
		t.zero += th
	}
	for t.zero > th {
		t.zero -= th
	}
}

func (t *Texture) Pitch() int {
	return t.Width() * 4
}

func (t *Texture) Width() int {
	_, _, w, _, _ := t.tex.Query()
	return w
}

func (t *Texture) Height() int {
	_, _, _, h, _ := t.tex.Query()
	return h
}

func (t *Texture) Render(ren *sdl.Renderer) error {
	zero := int32(t.zero)
	w := int32(t.Width())
	h := int32(t.Height())
	srcRect := &sdl.Rect{
		Y: zero,
		W: w,
		H: h - zero,
	}
	dstRect := &sdl.Rect{
		W: w,
		H: h - zero,
	}
	if ren.RenderCopy(t.tex, srcRect, dstRect) != 0 {
		return sdlError()
	}
	if zero > 0 {
		srcRect := &sdl.Rect{
			Y: 0,
			W: w,
			H: zero,
		}
		dstRect := &sdl.Rect{
			Y: h - zero,
			W: w,
			H: zero,
		}
		if ren.RenderCopy(t.tex, srcRect, dstRect) != 0 {
			return sdlError()
		}
	}
	return nil
}

func (t *Texture) Dispose() {
	if t.tex != nil {
		t.tex.Destroy()
		t.tex = nil
	}
}

type Pixel struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

func (p *Pixel) SetRGB(r, g, b uint8) {
	p.R, p.G, p.B = r, g, b
}

func (p *Pixel) GetRGB() (r, g, b uint8) {
	return p.R, p.G, p.B
}

/*
func loadBMPTexture(file string, ren *sdl.Renderer) (*sdl.Texture, error) {
	bmp := sdl.LoadBMP(file)
	if bmp == nil {
		return nil, sdlError()
	}
	defer bmp.Free()

	tex := ren.CreateTextureFromSurface(bmp)
	if tex == nil {
		return nil, sdlError()
	}
	return tex, nil
}

func renderText(message, fontFile string, color sdl.Color, fontSize int,
	renderer *sdl.Renderer) (*sdl.Texture, error) {
	font := ttf.OpenFont(fontFile, fontSize)
	if font == nil {
		return nil, sdlError()
	}
	defer font.Close()
	font.SetHinting(ttf.HintingLight)

	surf := font.RenderTextShaded(message, color, sdl.Color{})
	if surf == nil {
		return nil, sdlError()
	}
	defer surf.Free()

	texture := renderer.CreateTextureFromSurface(surf)
	if texture == nil {
		return nil, sdlError()
	}

	return texture, nil
}


	tex := renderText(Text,
		"/usr/share/fonts/truetype/ubuntu-font-family/UbuntuMono-R.ttf",
		sdl.Color{0x88, 0x88, 0x88, 255}, 16, ren)
	defer tex.Destroy()
*/

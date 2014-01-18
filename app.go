// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"go-sdl"
	//	"go-sdl/ttf"
	"log"
)

var (
	ResponseTime = 0.13 // seconds
	ScrollStep   = 100  // pixels
)

type App struct {
	m *Model
	v *View
}

func newApp() (*App, error) {
	if sdl.Init(sdl.InitVideo) != 0 {
		return nil, sdlError()
	}

	//	if ttf.Init() != 0 {
	//		return nil, sdlError()
	//	}
	m, err := newModel()
	if err != nil {
		return nil, err
	}

	v, err := newView(m)
	if err != nil {
		return nil, err
	}

	a := &App{m, v}
	return a, nil
}

func (a *App) refreshView() {
	a.v.Reload()
	a.v.Render()
}

func (a *App) Dispose() {
	if a.v != nil {
		a.v.Dispose()
		a.v = nil
	}
	if a.m != nil {
		a.m.Dispose()
		a.m = nil
	}

	//	ttf.Quit()
	sdl.Quit()
}

func (a *App) OpenDoc(file string) error {
	return a.m.Open(file)
}

func (a *App) Run() error {
	var e sdl.Event
	quit := false
	for !quit {
		if e.Wait() == 1 {
			switch e.Type() {
			case sdl.Keydown:
				switch e.Key().Keysym.Sym {
				case sdl.KEscape:
					quit = true
				case sdl.KDown:
					a.v.SmoothScroll(ScrollStep, ResponseTime)
				case sdl.KUp:
					a.v.SmoothScroll(-ScrollStep, ResponseTime)
				case sdl.KPagedown:
					step := a.v.Height() * 9 / 10
					a.v.SmoothScroll(step, ResponseTime)
				case sdl.KPageup:
					step := a.v.Height() * 9 / 10
					a.v.SmoothScroll(-step, ResponseTime)
				case sdl.KD:
					a.m.FlipDarkMode()
					a.refreshView()
				}
			case sdl.Quit_:
				quit = true
			case sdl.Windowevent:
				switch e.Window().Event {
				case sdl.WindoweventExposed:
					a.v.Render()
				case sdl.WindoweventResized:
					a.m.SetWidth(a.v.Width())
					a.refreshView()
				}
			}
		}
	}
	return nil
}

func logError(msg string) {
	log.Println(msg, "error:", sdl.GetError())
}

func sdlError() error {
	defer sdl.ClearError()
	if msg := sdl.GetError(); msg != "" {
		return errors.New(msg)
	}
	return nil
}

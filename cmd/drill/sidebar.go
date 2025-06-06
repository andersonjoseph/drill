package main

import (
	"github.com/andersonjoseph/drill/internal/components/breakpoints"
	"github.com/andersonjoseph/drill/internal/components/localvariables"
)

type sidebar struct {
	localVariables localvariables.Model
	breakpoints    breakpoints.Model
	width          int
	height         int
}

func (s *sidebar) calcSize(w, h int) {
	w = w / 2
	if w >= 50 {
		w = 50
	} else if w <= 20 {
		w = 20
	}
	s.width = w

	h = h / 4
	if h >= 15 {
		h = 15
	} else if h <= 3 {
		h = 3
	}
	s.height = h
}

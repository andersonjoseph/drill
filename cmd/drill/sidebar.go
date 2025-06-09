package main

import (
	"github.com/andersonjoseph/drill/internal/components/breakpoints"
	"github.com/andersonjoseph/drill/internal/components/callstack"
	"github.com/andersonjoseph/drill/internal/components/localvariables"
)

type sidebar struct {
	localVariables localvariables.Model
	breakpoints    breakpoints.Model
	callstack      callstack.Model
	errorMessage   errMsgModel
}

func (s *sidebar) calcSize(w, h int) (int, int) {
	w = w / 2
	if w >= 50 {
		w = 50
	} else if w <= 20 {
		w = 20
	}

	h = h / 4
	if h >= 10 {
		h = 10
	} else if h <= 3 {
		h = 3
	}

	return w, h
}

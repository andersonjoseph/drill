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

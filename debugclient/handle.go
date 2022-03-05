package debugclient

/*
#cgo LDFLAGS: -Llib/default/qmldap.2a21ec89 -lqmldap
#include <stdlib.h>
#include "lib/lib.h"
#include "bind.h"
*/
import "C"

import (
	"encoding/json"
	"unsafe"

	gopointer "github.com/mattn/go-pointer"
)

type Handle struct {
	Handle   *C.Handle
	Callback func(i json.RawMessage)
	self     unsafe.Pointer

	evaluate         chan json.RawMessage
	stackTrace       chan json.RawMessage
	scope            chan json.RawMessage
	lookup           chan json.RawMessage
	setBreakpoint    chan json.RawMessage
	changeBreakpoint chan json.RawMessage
	clearBreakpoint  chan json.RawMessage
}

//export goCallback
func goCallback(userData unsafe.Pointer, data *C.char) {
	gopointer.Restore(userData).(*Handle).callback(data)
}

func New() *Handle {
	h := &Handle{}
	h.self = gopointer.Save(h)
	h.Handle = C.makeLibraryHandle(h.self, C.callbackGo)
	h.evaluate = make(chan json.RawMessage)
	h.stackTrace = make(chan json.RawMessage)
	h.scope = make(chan json.RawMessage)
	h.lookup = make(chan json.RawMessage)
	h.setBreakpoint = make(chan json.RawMessage)
	h.changeBreakpoint = make(chan json.RawMessage)
	h.clearBreakpoint = make(chan json.RawMessage)
	return h
}
func (h *Handle) callback(s *C.char) {
	data := C.GoString(s)
	var r json.RawMessage
	err := json.Unmarshal([]byte(data), &r)
	if err != nil {
		panic(err)
	}

	var m map[string]interface{}
	feh := json.Unmarshal(r, &m)
	if feh != nil {
		return
	}

	if m["signal"] == "v4-result" {
		if m["command"] == "evaluate" {
			h.evaluate <- r
		} else if m["command"] == "backtrace" {
			h.stackTrace <- r
		} else if m["command"] == "scope" {
			h.scope <- r
		} else if m["command"] == "lookup" {
			h.lookup <- r
		} else if m["command"] == "setbreakpoint" {
			h.setBreakpoint <- r
		} else if m["command"] == "changebreakpoint" {
			h.changeBreakpoint <- r
		} else if m["command"] == "clearbreakpoint" {
			h.clearBreakpoint <- r
		}
	} else if m["signal"] == "v4-failure" {
		if m["command"] == "evaluate" {
			h.evaluate <- r
		} else if m["command"] == "backtrace" {
			h.stackTrace <- r
		} else if m["command"] == "scope" {
			h.scope <- r
		} else if m["command"] == "lookup" {
			h.lookup <- r
		} else if m["command"] == "setbreakpoint" {
			h.setBreakpoint <- r
		} else if m["command"] == "changebreakpoint" {
			h.changeBreakpoint <- r
		} else if m["command"] == "clearbreakpoint" {
			h.clearBreakpoint <- r
		}
	}

	h.Callback(r)
}
func (h *Handle) RunEventLoop() {
	C.execHandle(h.Handle)
}
func (h *Handle) invoke(i interface{}) json.RawMessage {
	data, feh := json.Marshal(i)
	if feh != nil {
		panic(feh)
	}

	cstr := C.CString(string(data))
	defer C.free(unsafe.Pointer(cstr))

	res := C.invokeHandle(h.Handle, cstr)
	retData := C.GoString(res)

	var r json.RawMessage
	feh = json.Unmarshal([]byte(retData), &r)
	if feh != nil {
		panic(feh)
	}

	return r
}
func (h *Handle) release() {
	C.freeLibraryHandle(h.Handle)
}

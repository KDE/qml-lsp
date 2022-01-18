package main

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

type handle struct {
	Handle   *C.Handle
	Callback func(i json.RawMessage)
	self     unsafe.Pointer
}

//export goCallback
func goCallback(userData unsafe.Pointer, data *C.char) {
	gopointer.Restore(userData).(*handle).callback(data)
}

func (h *handle) init() {
	h.self = gopointer.Save(h)
	h.Handle = C.makeLibraryHandle(h.self, C.callbackGo)
}
func (h *handle) callback(s *C.char) {
	data := C.GoString(s)
	var r json.RawMessage
	err := json.Unmarshal([]byte(data), &r)
	if err != nil {
		panic(err)
	}

	h.Callback(r)
}
func (h *handle) exec() {
	C.execHandle(h.Handle)
}
func (h *handle) invoke(i interface{}) json.RawMessage {
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
func (h *handle) release() {
	C.freeLibraryHandle(h.Handle)
}

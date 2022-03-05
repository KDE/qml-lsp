package main

import (
	"encoding/json"
	"qml-lsp/debugclient"
	"runtime"
)

type obj map[string]interface{}

func serverMain(h *debugclient.Handle) {
	StartServer(h)
}

func main() {
	h := debugclient.New()
	h.Callback = func(i json.RawMessage) {
		println(string(i))
	}

	initted := make(chan bool)

	go func() {
		<-initted

		runtime.LockOSThread()

		serverMain(h)
	}()

	runtime.LockOSThread()
	h.Init()
	initted <- true
	h.RunEventLoop()
}

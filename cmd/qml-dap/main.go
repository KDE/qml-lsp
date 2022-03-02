package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"qml-lsp/debugclient"
	"runtime"
)

type obj map[string]interface{}

func serverMain(h *debugclient.Handle) {
	StartServer(h)
}

func pront(s string) {
	cmd := exec.Command("systemd-cat")
	var b bytes.Buffer
	b.Write([]byte(s))
	cmd.Stdin = &b
	err := cmd.Start()

	if err != nil {
		panic(err)
	}

	err = cmd.Wait()
	if err != nil {
		panic(err)
	}
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

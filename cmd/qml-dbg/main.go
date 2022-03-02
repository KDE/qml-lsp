package main

import (
	"encoding/json"
	"fmt"
	"path"
	"qml-lsp/debugclient"
	"runtime"
	"strings"

	"github.com/chzyer/readline"
)

func appMain(handle *debugclient.Handle) {
	rl, err := readline.New("> ")
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	println(`Hi, welcome to qml-dbg! Type "help" if you want me to explain how you use me.`)

	connected := false

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF
			break
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		handle.Callback = func(i json.RawMessage) {
			var m map[string]interface{}
			feh := json.Unmarshal(i, &m)
			if feh != nil {
				return
			}

			if m["signal"] == "disconnected" {
				println("\nLooks like the program you're debugging exited unexpectedly...")
				connected = false
			} else if m["signal"] == "connected" {
				println("\nI connected to the program! Now, you can start debugging it.")
				connected = true
			} else if m["signal"] == "v4-stopped" {
				println("\nThe program stopped because it threw an exception!")
				println("Run 'backtrace' to see a stack trace.")
			}
		}

		switch fields[0] {
		case "attach":
			if len(fields) < 2 {
				println("Not enough arguments; did you forget to tell me what to attach to?")
				println("You can tell me to attach to something like this:")
				println("\n\tattach localhost:5050\n")
				println("You can make a QML program ready to be attached to by adding")
				println("'-qmljsdebugger=port:5050,block' to its command-line arguments.")
				println("For example, running")
				println("\n\tneochat -qmljsdebugger=port:5050,block\n")
				println("will open a new NeoChat process and have it wait on me to attach to it.")
				println("If you want me to launch the program myself, try using the 'launch' command.")
				continue
			}
			println("Connecting to " + fields[1] + "...")
			handle.Connect(fields[1])
		case "backtrace", "bt":
			if !connected {
				println("I'm not debugging a program currently. Did you mean to 'attach' to one first?")
				continue
			}
			frames, err := handle.Backtrace()
			if err != nil {
				println(fmt.Sprintf("Sorry, I ran into an error while getting the backtrace!\n\t%s", err))
			}
			println("Most recently called function:")
			for i := len(frames.Body.Frames) - 1; i >= 0; i-- {
				frame := frames.Body.Frames[i]

				println(fmt.Sprintf("\t%s in %s:%d (%s)", frame.Func, path.Base(frame.Script), frame.Line, frame.Script))
			}
		case "continue", "cont", "c":
			if !connected {
				println("I'm not debugging a program currently. Did you mean to 'attach' to one first?")
				continue
			}
			handle.Continue()
		case "stepin", "in", "i":
			if !connected {
				println("I'm not debugging a program currently. Did you mean to 'attach' to one first?")
				continue
			}
			handle.StepIn()
		case "stepout", "out", "o":
			if !connected {
				println("I'm not debugging a program currently. Did you mean to 'attach' to one first?")
				continue
			}
			handle.StepOut()
		case "stepnext", "next", "n":
			if !connected {
				println("I'm not debugging a program currently. Did you mean to 'attach' to one first?")
				continue
			}
			handle.StepNext()
		default:
			println("Sorry, I don't recognise that command.\n")
			println("Did you mean to eval that?")
		}
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

		appMain(h)
	}()

	runtime.LockOSThread()
	h.Init()
	initted <- true
	h.RunEventLoop()
}

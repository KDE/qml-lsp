package main

import (
	"encoding/json"
	"fmt"
	"path"
	"qml-lsp/debugclient"
	"runtime"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
)

type bpointData struct {
	file    string
	line    int
	enabled bool
}

func appMain(handle *debugclient.Handle) {
	rl, err := readline.New("> ")
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	println(`Hi, welcome to qml-dbg! Type "help" if you want me to explain how you use me.`)

	connected := false
	breakpoints := map[int]bpointData{}

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
				println("\nThe program paused!")
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
		case "eval":
			if !connected {
				println("I'm not debugging a program currently. Did you mean to 'attach' to one first?")
				continue
			}
			script := strings.TrimPrefix(strings.TrimSpace(line), "eval")
			val, err, ok := handle.Evaluate(script)
			if err != nil {
				println("Error evaluating:", err.Error())
				continue
			}
			if !ok {
				println("Your script was valid, but threw an error while executing.")
				continue
			}
			println(val.String())
		case "breakpoints", "bs":
			if !connected {
				println("I'm not debugging a program currently. Did you mean to 'attach' to one first?")
				continue
			}
			println("Breakpoints:")
			for num, bp := range breakpoints {
				if bp.enabled {
					println(fmt.Sprintf("\t#%d at %s:%d", num, bp.file, bp.line))
				} else {
					println(fmt.Sprintf("\t(disabled) #%d at %s:%d", num, bp.file, bp.line))
				}
			}
		case "breakpoint-disable", "bd":
			if !connected {
				println("I'm not debugging a program currently. Did you mean to 'attach' to one first?")
				continue
			}
			if len(fields) < 2 {
				println("You need to provide me with the breakpoint number to disable.")
				println("You can see which breakpoints exist with 'breakpoints'")
				continue
			}
			nnum, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				println("I couldn't disable your breakpoint because I didn't understand your number:", err.Error())
				continue
			}
			num := int(nnum)
			k, ok := breakpoints[num]
			if !ok {
				println("Breakpoint", num, "doesn't exist")
				continue
			}
			if !k.enabled {
				println("Breakpoint", num, "is already disabled")
			}
			_, feh := handle.SetBreakpointEnabled(int(num), false)
			if feh != nil {
				println("I had an issue while disabling your breakpoint:", feh.Error())
				continue
			}
			k.enabled = false
			breakpoints[num] = k
			println("Disabled breakpoint", num)
		case "breakpoint-enable", "be":
			if !connected {
				println("I'm not debugging a program currently. Did you mean to 'attach' to one first?")
				continue
			}
			if len(fields) < 2 {
				println("You need to provide me with the breakpoint number to disable.")
				println("You can see which breakpoints exist with 'breakpoints'")
				continue
			}
			nnum, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				println("I couldn't disable your breakpoint because I didn't understand your number:", err.Error())
				continue
			}
			num := int(nnum)
			k, ok := breakpoints[num]
			if !ok {
				println("Breakpoint", num, "doesn't exist")
				continue
			}
			if k.enabled {
				println("Breakpoint", num, "is already enabled")
			}
			_, feh := handle.SetBreakpointEnabled(int(num), true)
			if feh != nil {
				println("I had an issue while enabling your breakpoint:", feh.Error())
				continue
			}
			k.enabled = true
			breakpoints[num] = k
			println("Enabled breakpoint", num)
		case "breakpoint", "b":
			if !connected {
				println("I'm not debugging a program currently. Did you mean to 'attach' to one first?")
				continue
			}
			switch len(fields) {
			case 1:
				println("You need to provide a place to breakpoint.\nA breakpoint place looks like this:" +
					"\n\tButton.qml:25")
			case 2:
				k := strings.Split(fields[1], ":")
				switch len(k) {
				case 0, 1:
					println("You need to provide a file and line number like this:" +
						"\n\tButton.qml:25")
				case 2:
					line, err := strconv.ParseInt(k[1], 10, 64)
					if err != nil {
						println("I couldn't set your breakpoint because I didn't understand your line number:", err.Error())
						continue
					}

					breakpoint, err := handle.SetBreakpoint(k[0], int(line))
					if err != nil {
						println("I had an issue setting your breakpoint:", err.Error())
						continue
					}

					println(fmt.Sprintf("I set breakpoint #%d", breakpoint.ID))
					println("The program will pause right before it starts to run that line of code")
					println(fmt.Sprintf("You can disable it with 'breakpoint-disable %d'", breakpoint.ID))
					println("See active breakpoints with 'breakpoints'")

					breakpoints[breakpoint.ID] = bpointData{
						file:    k[0],
						line:    int(line),
						enabled: true,
					}
				}
			}
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

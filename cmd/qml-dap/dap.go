package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-dap"
)

type server struct {
	h *handle

	evaluate   chan json.RawMessage
	stackTrace chan json.RawMessage
	scope      chan json.RawMessage
	lookup     chan json.RawMessage

	frames map[int]Frames

	launchData struct {
		name   string
		method string
		pid    int
	}
}

type connection struct {
}

func (c *connection) Notify(event string, d dap.EventMessage) {
	d.GetEvent().Event = event
	d.GetEvent().Type = "event"
	dap.WriteProtocolMessage(os.Stdout, d)
}

func (s *server) qmlDbgCallback(i json.RawMessage) {
	c := connection{}

	var m map[string]interface{}
	feh := json.Unmarshal(i, &m)
	if feh != nil {
		return
	}

	if m["signal"] == "v4-stopped" {
		c.Notify("stopped", &dap.StoppedEvent{Body: dap.StoppedEventBody{
			Reason:            "exception",
			ThreadId:          1,
			AllThreadsStopped: true,
		}})
	} else if m["signal"] == "v4-result" {
		if m["command"] == "evaluate" {
			s.evaluate <- i
		} else if m["command"] == "backtrace" {
			s.stackTrace <- i
		} else if m["command"] == "scope" {
			s.scope <- i
		} else if m["command"] == "lookup" {
			s.lookup <- i
		}
	} else if m["signal"] == "v4-failure" {
		if m["command"] == "evaluate" {
			s.evaluate <- i
		} else if m["command"] == "backtrace" {
			s.stackTrace <- i
		} else if m["command"] == "scope" {
			s.scope <- i
		} else if m["command"] == "lookup" {
			s.lookup <- i
		}
	} else if m["signal"] == "v4-connected" {
		c.Notify("terminated", &dap.ProcessEvent{
			Body: dap.ProcessEventBody{
				Name:            s.launchData.name,
				StartMethod:     s.launchData.method,
				SystemProcessId: s.launchData.pid,
			},
		})
	} else if m["signal"] == "disconnected" {
		c.Notify("terminated", &dap.TerminatedEvent{})
	}
}

func (s *server) Initialize(ctx context.Context, conn *connection, params *dap.InitializeRequest) (*dap.InitializeResponse, error) {
	conn.Notify("initialized", &dap.InitializedEvent{})

	s.h.Callback = s.qmlDbgCallback
	s.frames = map[int]Frames{}
	s.evaluate = make(chan json.RawMessage)
	s.stackTrace = make(chan json.RawMessage)
	s.scope = make(chan json.RawMessage)
	s.lookup = make(chan json.RawMessage)

	return &dap.InitializeResponse{
		Body: dap.Capabilities{},
	}, nil
}

func (s *server) SetBreakpoints(ctx context.Context, conn *connection, params *dap.SetBreakpointsRequest) (*dap.SetBreakpointsResponse, error) {
	return &dap.SetBreakpointsResponse{}, nil
}

func (s *server) Continue(
	ctx context.Context, conn *connection,
	params *dap.ContinueRequest) (*dap.ContinueResponse, error) {

	s.h.invoke(obj{"method": "continue", "kind": "continue"})

	return &dap.ContinueResponse{}, nil
}

func (s *server) Pause(
	ctx context.Context, conn *connection,
	params *dap.PauseRequest) (*dap.PauseResponse, error) {

	s.h.invoke(obj{"method": "interrupt"})

	return &dap.PauseResponse{}, nil
}

func (s *server) Scopes(
	ctx context.Context, conn *connection,
	params *dap.ScopesRequest) (*dap.ScopesResponse, error) {

	frame := s.frames[params.Arguments.FrameId]

	kinds := map[int]string{
		ScopeGlobal:  "global",
		ScopeLocal:   "local",
		ScopeWith:    "with",
		ScopeClosure: "closure",
		ScopeCatch:   "catch",
	}

	r := &dap.ScopesResponse{}
	for _, scope := range frame.Scopes {

		r.Body.Scopes = append(r.Body.Scopes, dap.Scope{
			Name:               fmt.Sprintf("%d %s", scope.Index, kinds[scope.Type]),
			VariablesReference: (frame.Index << 8) | scope.Index,
			Expensive:          false,
		})
	}

	return r, nil
}

func (s *server) Next(
	ctx context.Context, conn *connection,
	params *dap.NextRequest) (*dap.NextResponse, error) {

	s.h.invoke(obj{"method": "continue", "kind": "next"})

	return &dap.NextResponse{}, nil
}

func (s *server) StepIn(
	ctx context.Context, conn *connection,
	params *dap.StepInRequest) (*dap.StepInResponse, error) {

	s.h.invoke(obj{"method": "continue", "kind": "in"})

	return &dap.StepInResponse{}, nil
}

func (s *server) StepOut(
	ctx context.Context, conn *connection,
	params *dap.StepOutRequest) (*dap.StepOutResponse, error) {

	s.h.invoke(obj{"method": "continue", "kind": "out"})

	return &dap.StepOutResponse{}, nil
}

func (s *server) Evaluate(
	ctx context.Context, conn *connection,
	params *dap.EvaluateRequest) (*dap.EvaluateResponse, error) {

	s.h.invoke(obj{"method": "eval", "script": params.Arguments.Expression})

	msg := <-s.evaluate

	var m struct {
		Signal string   `json:"signal"`
		Body   QMLValue `json:"body"`
	}
	json.Unmarshal(msg, &m)

	if m.Signal == "v4-failure" {
		return nil, errors.New("eval failed")
	}

	ref := 0
	if m.Body.Ref != 0 && m.Body.Type != "function" {
		ref = (1 << 16) | m.Body.Ref
	}

	return &dap.EvaluateResponse{
		Body: dap.EvaluateResponseBody{
			Result:             m.Body.String(),
			VariablesReference: ref,
		},
	}, nil
}

func (s *server) Threads(
	ctx context.Context, conn *connection,
	params *dap.ThreadsRequest) (*dap.ThreadsResponse, error) {

	return &dap.ThreadsResponse{
		Body: dap.ThreadsResponseBody{
			Threads: []dap.Thread{
				{
					Id:   1,
					Name: "QML Thread",
				},
			},
		},
	}, nil
}

func (s *server) SetExceptionBreakpoints(
	ctx context.Context, conn *connection,
	params *dap.SetExceptionBreakpointsRequest) (*dap.SetExceptionBreakpointsResponse, error) {

	return &dap.SetExceptionBreakpointsResponse{}, nil
}

type StackFramesResponse struct {
	Body    StackFramesBody `json:"body"`
	Command string          `json:"command"`
	Signal  string          `json:"signal"`
}
type Scopes struct {
	Index int `json:"index"`
	Type  int `json:"type"`
}
type Frames struct {
	DebuggerFrame bool     `json:"debuggerFrame"`
	Func          string   `json:"func"`
	Index         int      `json:"index"`
	Line          int      `json:"line"`
	Scopes        []Scopes `json:"scopes"`
	Script        string   `json:"script"`
}
type StackFramesBody struct {
	Frames    []Frames `json:"frames"`
	FromFrame int      `json:"fromFrame"`
	ToFrame   int      `json:"toFrame"`
}

type ScopeResponse struct {
	Body    ScopeBody `json:"body"`
	Command string    `json:"command"`
	Signal  string    `json:"signal"`
}

type ScopeBody struct {
	FrameIndex int      `json:"frameIndex"`
	Index      int      `json:"index"`
	Object     QMLValue `json:"object"`
	Type       int      `json:"type"`
}

type LookupResponse struct {
	Body    map[string]QMLValue `json:"body"`
	Command string              `json:"command"`
	Signal  string              `json:"signal"`
}
type QMLValue struct {
	Type       string          `json:"type"`
	Handle     int             `json:"handle"`
	Value      json.RawMessage `json:"value"`
	Ref        int             `json:"ref,omitempty"`
	Name       string          `json:"name,omitempty"`
	Properties []QMLValue      `json:"properties,omitempty"`
}

func (v *QMLValue) String() string {
	switch v.Type {
	case "undefined":
		return "undefined"
	case "null":
		return "null"
	case "boolean":
		return string(v.Value)
	case "number":
		return string(v.Value)
	case "string":
		return string(v.Value)
	case "object":
		return "[object Object]"
	case "function":
		return fmt.Sprintf("[function %s]", v.Name)
	case "script":
		return fmt.Sprintf("[script %s]", v.Name)
	default:
		return fmt.Sprintf("[%s %s]", v.Type, string(v.Value))
	}
}

const (
	ScopeGlobal  = 0x0
	ScopeLocal   = 0x1
	ScopeWith    = 0x2
	ScopeClosure = 0x3
	ScopeCatch   = 0x4
)

func (s *server) Variables(
	ctx context.Context, conn *connection,
	params *dap.VariablesRequest) (*dap.VariablesResponse, error) {

	ref := params.Arguments.VariablesReference
	isHandle := ref >> 16 & 0xFF
	frame := ref >> 8 & 0xFF
	scope := ref & 0xFF

	if isHandle != 0 {
		handle := ref & 0xFF

		s.h.invoke(obj{
			"method":         "lookup",
			"include-source": true,
			"handles":        []int{handle},
		})

		lmsg := <-s.lookup
		var lresponse LookupResponse

		feh := json.Unmarshal(lmsg, &lresponse)
		if feh != nil {
			return nil, fmt.Errorf("failed to get variables: %w", feh)
		}

		shandle := strconv.FormatInt(int64(handle), 10)

		var variables []dap.Variable
		for _, variable := range lresponse.Body[shandle].Properties {
			ref := 0
			if variable.Ref != 0 {
				ref = (1 << 16) | variable.Ref
			}
			if variable.Type == "function" {
				ref = 0
			}
			variables = append(variables, dap.Variable{
				Name:               variable.Name,
				Value:              variable.String(),
				Type:               variable.Type,
				VariablesReference: ref,
			})
		}

		return &dap.VariablesResponse{
			Body: dap.VariablesResponseBody{
				Variables: variables,
			},
		}, nil
	}

	s.h.invoke(obj{
		"method":       "scope",
		"frame-number": frame,
		"scope-number": scope,
	})

	msg := <-s.scope
	var response ScopeResponse

	feh := json.Unmarshal(msg, &response)
	if feh != nil {
		return nil, fmt.Errorf("failed to get variables: %w", feh)
	}

	s.h.invoke(obj{
		"method":         "lookup",
		"include-source": true,
		"handles":        []int{response.Body.Object.Handle},
	})

	lmsg := <-s.lookup
	var lresponse LookupResponse

	feh = json.Unmarshal(lmsg, &lresponse)
	if feh != nil {
		return nil, fmt.Errorf("failed to get variables: %w", feh)
	}

	var variables []dap.Variable

	shandle := strconv.FormatInt(int64(response.Body.Object.Handle), 10)
	for _, variable := range lresponse.Body[shandle].Properties {
		ref := 0
		if variable.Ref != 0 {
			ref = (1 << 16) | variable.Ref
		}
		if variable.Type == "function" {
			ref = 0
		}
		variables = append(variables, dap.Variable{
			Name:               variable.Name,
			Value:              variable.String(),
			Type:               variable.Type,
			VariablesReference: ref,
		})
	}

	return &dap.VariablesResponse{
		Body: dap.VariablesResponseBody{
			Variables: variables,
		},
	}, nil
}

func (s *server) StackTrace(
	ctx context.Context, conn *connection,
	params *dap.StackTraceRequest) (*dap.StackTraceResponse, error) {

	s.h.invoke(obj{"method": "backtrace"})

	msg := <-s.stackTrace

	var response StackFramesResponse

	feh := json.Unmarshal(msg, &response)
	if feh != nil {
		return nil, fmt.Errorf("failed to get stack trace: %w", feh)
	}

	var frames []dap.StackFrame

	for _, frame := range response.Body.Frames {
		if strings.HasPrefix(frame.Script, "file://") {
			frame.Script = strings.TrimPrefix(frame.Script, "file://")
		}
		s.frames[frame.Index] = frame
		frames = append(frames, dap.StackFrame{
			Id:   frame.Index,
			Name: frame.Func,
			Line: frame.Line + 1,
			Source: dap.Source{
				Name: path.Base(frame.Script),
				Path: frame.Script,
			},
		})
	}

	return &dap.StackTraceResponse{
		Body: dap.StackTraceResponseBody{
			StackFrames: frames,
			TotalFrames: len(frames),
		},
	}, nil
}

func (s *server) Attach(
	ctx context.Context, conn *connection,
	params *dap.AttachRequest) (*dap.AttachResponse, error) {

	var t struct {
		Target string `json:"target"`
	}
	json.Unmarshal(params.Arguments, &t)

	s.launchData.name = t.Target
	s.launchData.pid = 0
	s.launchData.method = "attach"

	s.h.invoke(obj{"method": "connect", "target": t.Target})

	return &dap.AttachResponse{}, nil
}

func (s *server) Launch(ctx context.Context, conn *connection, params *dap.LaunchRequest) (*dap.LaunchResponse, error) {
	var t struct {
		Program  string `json:"program"`
		QMLScene bool   `json:"qmlscene"`
	}
	json.Unmarshal(params.Arguments, &t)

	arg := "-qmljsdebugger=port:5050,block"
	var cmd *exec.Cmd
	if t.QMLScene {
		cmd = exec.Command("qmlscene", t.Program, arg)
	} else {
		cmd = exec.Command(t.Program, arg)
	}
	feh := cmd.Start()
	if feh != nil {
		return nil, feh
	}

	s.launchData.name = path.Base(t.Program)
	s.launchData.pid = cmd.Process.Pid
	s.launchData.method = "launch"

	time.Sleep(1 * time.Second)

	s.h.invoke(obj{"method": "connect", "target": "localhost:5050"})

	return &dap.LaunchResponse{}, nil
}

type method func(context.Context, *connection, dap.Message)
type methodmap map[string]method

func zu(fn interface{}) method {
	val := reflect.ValueOf(fn)

	return func(ctx context.Context, conn *connection, params dap.Message) {
		ret := val.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(conn), reflect.ValueOf(params)})

		switch len(ret) {
		case 0: // notification
		case 2: // dap
			req := params.(dap.RequestMessage).GetRequest()

			if ret[0].IsNil() {
				if ret[1].IsNil() {
					panic("nil return")
				}

				er := dap.ErrorResponse{}
				er.Type = "response"
				er.Command = req.Command
				er.Success = false
				er.RequestSeq = req.Seq
				er.Message = ret[1].Interface().(error).Error()
				er.Body.Error.Format = ret[1].Interface().(error).Error()
				er.Body.Error.ShowUser = true

				dap.WriteProtocolMessage(os.Stdout, &er)
				return
			}

			resp := ret[0].Interface().(dap.ResponseMessage)
			rresp := resp.GetResponse()
			rresp.Type = "response"
			rresp.Success = true
			rresp.Command = req.Command
			rresp.RequestSeq = req.Seq

			dap.WriteProtocolMessage(os.Stdout, resp)
		default:
			panic("unknown arity of return")
		}
	}
}

func StartServer(h *handle) {
	reader := bufio.NewReader(os.Stdin)
	s := server{h: h}
	a := methodmap{
		"initialize":              zu(s.Initialize),
		"launch":                  zu(s.Launch),
		"setBreakpoints":          zu(s.SetBreakpoints),
		"continue":                zu(s.Continue),
		"next":                    zu(s.Next),
		"stepIn":                  zu(s.StepIn),
		"stepOut":                 zu(s.StepOut),
		"pause":                   zu(s.Pause),
		"stackTrace":              zu(s.StackTrace),
		"evaluate":                zu(s.Evaluate),
		"scopes":                  zu(s.Scopes),
		"variables":               zu(s.Variables),
		"threads":                 zu(s.Threads),
		"attach":                  zu(s.Attach),
		"setExceptionBreakpoints": zu(s.SetExceptionBreakpoints),
	}

	for {
		req, feh := dap.ReadProtocolMessage(reader)

		if feh != nil {
			continue
		}

		ctx := context.Background()

		switch request := req.(type) {
		case dap.RequestMessage:
			r := request.GetRequest()
			w, ok := a[r.Command]
			if !ok {
				continue
			}
			w(ctx, &connection{}, request)
		default:
		}
	}
}

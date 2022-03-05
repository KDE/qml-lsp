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
	"qml-lsp/debugclient"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-dap"
)

type server struct {
	h *debugclient.Handle

	frames      map[int]debugclient.Frames
	breakpoints map[string][]int

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
	} else if m["signal"] == "disconnected" {
		c.Notify("terminated", &dap.ProcessEvent{
			Body: dap.ProcessEventBody{
				Name:            s.launchData.name,
				StartMethod:     s.launchData.method,
				SystemProcessId: s.launchData.pid,
			},
		})
	}
}

func (s *server) Initialize(ctx context.Context, conn *connection, params *dap.InitializeRequest) (*dap.InitializeResponse, error) {
	conn.Notify("initialized", &dap.InitializedEvent{})

	s.h.Callback = s.qmlDbgCallback
	s.frames = map[int]debugclient.Frames{}
	s.breakpoints = map[string][]int{}

	return &dap.InitializeResponse{
		Body: dap.Capabilities{},
	}, nil
}

func (s *server) SetBreakpoints(ctx context.Context, conn *connection, params *dap.SetBreakpointsRequest) (*dap.SetBreakpointsResponse, error) {
	path := params.Arguments.Source.Path
	if v, ok := s.breakpoints[path]; ok {
		for _, bpoint := range v {
			_, feh := s.h.ClearBreakpoint(bpoint)
			if feh != nil {
				return nil, fmt.Errorf("failed to clear breakpoints before setting breakpoints: %+w", feh)
			}
		}
	}
	breakpointIDs := []int{}
	resp := &dap.SetBreakpointsResponse{}
	for _, it := range params.Arguments.Breakpoints {
		bpoint, feh := s.h.SetBreakpoint(path, it.Line)
		if feh != nil {
			return nil, fmt.Errorf("failed to set breakpoints: %+w", feh)
		}
		breakpointIDs = append(breakpointIDs, bpoint.ID)
		resp.Body.Breakpoints = append(resp.Body.Breakpoints, dap.Breakpoint{
			Id:     bpoint.ID,
			Source: params.Arguments.Source,
			Line:   it.Line,
		})
	}
	s.breakpoints[path] = breakpointIDs
	return resp, nil
}

func (s *server) Continue(
	ctx context.Context, conn *connection,
	params *dap.ContinueRequest) (*dap.ContinueResponse, error) {

	s.h.Continue()

	return &dap.ContinueResponse{}, nil
}

func (s *server) Pause(
	ctx context.Context, conn *connection,
	params *dap.PauseRequest) (*dap.PauseResponse, error) {

	s.h.Interrupt()

	return &dap.PauseResponse{}, nil
}

func (s *server) Scopes(
	ctx context.Context, conn *connection,
	params *dap.ScopesRequest) (*dap.ScopesResponse, error) {

	frame := s.frames[params.Arguments.FrameId]

	kinds := map[int]string{
		debugclient.ScopeGlobal:  "global",
		debugclient.ScopeLocal:   "local",
		debugclient.ScopeWith:    "with",
		debugclient.ScopeClosure: "closure",
		debugclient.ScopeCatch:   "catch",
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

	s.h.StepNext()

	return &dap.NextResponse{}, nil
}

func (s *server) StepIn(
	ctx context.Context, conn *connection,
	params *dap.StepInRequest) (*dap.StepInResponse, error) {

	s.h.StepIn()

	return &dap.StepInResponse{}, nil
}

func (s *server) StepOut(
	ctx context.Context, conn *connection,
	params *dap.StepOutRequest) (*dap.StepOutResponse, error) {

	s.h.StepNext()

	return &dap.StepOutResponse{}, nil
}

func (s *server) Evaluate(
	ctx context.Context, conn *connection,
	params *dap.EvaluateRequest) (*dap.EvaluateResponse, error) {

	m, err, ok := s.h.Evaluate(params.Arguments.Expression)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate: %+w", err)
	}
	if !ok {
		return nil, errors.New("eval failed")
	}

	ref := 0
	if m.Ref != 0 && m.Type != "function" {
		ref = (1 << 16) | m.Ref
	}

	return &dap.EvaluateResponse{
		Body: dap.EvaluateResponseBody{
			Result:             m.String(),
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

func (s *server) Variables(
	ctx context.Context, conn *connection,
	params *dap.VariablesRequest) (*dap.VariablesResponse, error) {

	ref := params.Arguments.VariablesReference
	isHandle := ref >> 16 & 0xFF
	frame := ref >> 8 & 0xFF
	scope := ref & 0xFF

	if isHandle != 0 {
		handle := ref & 0xFF

		lresponse, feh := s.h.Lookup(true, handle)
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

	response, feh := s.h.Scope(frame, scope)
	if feh != nil {
		return nil, fmt.Errorf("failed to get variables: %w", feh)
	}

	lresponse, feh := s.h.Lookup(true, response.Body.Object.Handle)
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

	response, feh := s.h.Backtrace()
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

	s.h.Connect(t.Target)

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
		return nil, fmt.Errorf("failed to start launch command: %+w", feh)
	}

	s.launchData.name = path.Base(t.Program)
	s.launchData.pid = cmd.Process.Pid
	s.launchData.method = "launch"

	time.Sleep(1 * time.Second)

	s.h.Connect("localhost:5050")

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

func StartServer(h *debugclient.Handle) {
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

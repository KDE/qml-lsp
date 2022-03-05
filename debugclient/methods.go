package debugclient

import (
	"encoding/json"
	"fmt"
)

type obj map[string]interface{}

func (h *Handle) Continue() {
	h.invoke(obj{"method": "continue", "kind": "continue"})
}

func (h *Handle) StepIn() {
	h.invoke(obj{"method": "continue", "kind": "in"})
}

func (h *Handle) StepOut() {
	h.invoke(obj{"method": "continue", "kind": "out"})
}

func (h *Handle) StepNext() {
	h.invoke(obj{"method": "continue", "kind": "next"})
}

func (h *Handle) Init() {
	h.invoke(obj{"method": "init"})
}

func (h *Handle) Interrupt() {
	h.invoke(obj{"method": "interrupt"})
}

func (h *Handle) Evaluate(s string) (ret QMLValue, err error, okay bool) {
	h.invoke(obj{"method": "eval", "script": s})

	msg := <-h.evaluate

	var m struct {
		Signal string   `json:"signal"`
		Body   QMLValue `json:"body"`
	}
	feh := json.Unmarshal(msg, &m)
	if feh != nil {
		return QMLValue{}, fmt.Errorf("failed to decode evaluate response: %+w", err), false
	}

	if m.Signal == "v4-failure" {
		return m.Body, nil, false
	}

	return m.Body, nil, true
}

func (h *Handle) SetBreakpointEnabled(num int, enabled bool) (ChangeBreakpointResponse, error) {
	h.invoke(obj{
		"method":  "set-breakpoint-enabled",
		"number":  num,
		"enabled": enabled,
	})

	msg := <-h.changeBreakpoint

	var m struct {
		Body ChangeBreakpointResponse `json:"body"`
	}
	feh := json.Unmarshal(msg, &m)
	if feh != nil {
		return ChangeBreakpointResponse{}, fmt.Errorf("failed to decode breakpoint response: %+w", feh)
	}

	return m.Body, nil
}

func (h *Handle) SetBreakpoint(file string, line int) (SetBreakpointResponse, error) {
	h.invoke(obj{
		"method": "set-breakpoint",
		"file":   file,
		"line":   line,
	})

	msg := <-h.setBreakpoint

	var m struct {
		Body SetBreakpointResponse `json:"body"`
	}
	feh := json.Unmarshal(msg, &m)
	if feh != nil {
		return SetBreakpointResponse{}, fmt.Errorf("failed to decode breakpoint response: %+w", feh)
	}

	return m.Body, nil
}

func (h *Handle) Lookup(includeSource bool, handles ...int) (LookupResponse, error) {
	h.invoke(obj{
		"method":         "lookup",
		"include-source": true,
		"handles":        handles,
	})

	lmsg := <-h.lookup
	var lresponse LookupResponse

	feh := json.Unmarshal(lmsg, &lresponse)
	if feh != nil {
		return LookupResponse{}, fmt.Errorf("failed to get variables: %w", feh)
	}

	return lresponse, nil
}

func (h *Handle) Scope(frame, scope int) (ScopeResponse, error) {
	h.invoke(obj{
		"method":       "scope",
		"frame-number": frame,
		"scope-number": scope,
	})

	msg := <-h.scope
	var response ScopeResponse

	feh := json.Unmarshal(msg, &response)
	if feh != nil {
		return ScopeResponse{}, fmt.Errorf("failed to get scope: %w", feh)
	}

	return response, nil
}

func (h *Handle) Backtrace() (StackFramesResponse, error) {
	h.invoke(obj{"method": "backtrace"})

	msg := <-h.stackTrace
	var response StackFramesResponse

	feh := json.Unmarshal(msg, &response)
	if feh != nil {
		return StackFramesResponse{}, fmt.Errorf("failed to get stack trace: %w", feh)
	}

	return response, nil
}

func (h *Handle) Connect(target string) {
	h.invoke(obj{"method": "connect", "target": target})
}

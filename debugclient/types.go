package debugclient

import (
	"encoding/json"
	"fmt"
)

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

type SetBreakpointResponse struct {
	ID   int    `json:"breakpoint"`
	Type string `json:"type"`
}

type ChangeBreakpointResponse struct {
	ID   int    `json:"breakpoint"`
	Type string `json:"type"`
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

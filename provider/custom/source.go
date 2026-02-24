// Package custom provides a bridge between the Go core and Lua-based scraper scripts.
package custom

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

type luaSource struct {
	name  string
	state *lua.LState
}

// Name returns the provider name.
func (s *luaSource) Name() string {
	return s.name
}

// ID returns the provider ID.
func (s *luaSource) ID() string {
	return IDfromName(s.name) // Defined in loader.go
}

func newLuaSource(name string, state *lua.LState) (*luaSource, error) {
	s := &luaSource{
		name:  name,
		state: state,
	}

	return s, nil
}

// call executes a global Lua function safely.
func (s *luaSource) call(fn string, retType lua.LValueType, args ...lua.LValue) (lua.LValue, error) {
	luaFn := s.state.GetGlobal(fn)
	if luaFn.Type() != lua.LTFunction {
		return nil, fmt.Errorf("function %s is not defined", fn)
	}

	err := s.state.CallByParam(lua.P{
		Fn:      luaFn,
		NRet:    1,
		Protect: true,
	}, args...)

	if err != nil {
		return nil, err
	}

	retval := s.state.Get(-1)
	s.state.Pop(1) // Clean stack

	if retval.Type() != retType {
		return nil, fmt.Errorf("%s returned %s, expected %s", fn, retval.Type(), retType)
	}

	return retval, nil
}

package custom

import (
	"fmt"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

type luaSource struct {
	name  string
	state *lua.LState
	mu    sync.Mutex // Protects the non-thread-safe Lua stack from concurrent access
}

func (s *luaSource) Name() string {
	return s.name
}

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

func (s *luaSource) call(fn string, retType lua.LValueType, args ...lua.LValue) (lua.LValue, error) {
	// Acquire lock to prevent stack corruption from concurrent TUI background updates.
	s.mu.Lock()
	defer s.mu.Unlock()

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

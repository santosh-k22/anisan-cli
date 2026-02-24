// Package scraper provides high-level coordination and execution for virtualized Lua-based scraping modules.
package scraper

import (
	"os"
	"sync"

	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

var bytecodeCache sync.Map

// PreCompileAndLoad executes a Lua script within the provided LState, utilizing a bytecode cache to minimize compilation overhead.
func PreCompileAndLoad(L *lua.LState, scriptPath string) error {
	// Check for cached prototype
	if cachedProto, exists := bytecodeCache.Load(scriptPath); exists {
		// Cache hit: Load pre-compiled bytecode prototype directly into the state.
		fn := L.NewFunctionFromProto(cachedProto.(*lua.FunctionProto))
		L.Push(fn)
		return L.PCall(0, lua.MultRet, nil)
	}

	// Cache miss: Parse the script and compile it into a reusable bytecode prototype.
	file, err := os.Open(scriptPath)
	if err != nil {
		return err
	}
	defer file.Close()

	chunk, err := parse.Parse(file, scriptPath)
	if err != nil {
		return err
	}

	proto, err := lua.Compile(chunk, scriptPath)
	if err != nil {
		return err
	}

	// Persist the bytecode prototype in the global cache for future re-execution.
	bytecodeCache.Store(scriptPath, proto)

	fn := L.NewFunctionFromProto(proto)
	L.Push(fn)
	return L.PCall(0, lua.MultRet, nil)
}

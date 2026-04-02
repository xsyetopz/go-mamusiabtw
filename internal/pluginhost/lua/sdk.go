package luaplugin

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"github.com/xsyetopz/go-mamusiabtw/internal/integrations/kawaii"
)

func (v *VM) luaPlugin(l *lua.LState) int {
	spec := l.CheckTable(1)
	l.Push(spec)
	return 1
}

func (v *VM) luaCommand(l *lua.LState) int {
	name := strings.TrimSpace(l.CheckString(1))
	spec := copyLuaTable(l, l.CheckTable(2))
	if name == "" {
		l.RaiseError("command name is required")
		return 0
	}
	spec.RawSetString("name", lua.LString(name))
	l.Push(spec)
	return 1
}

func (v *VM) luaJob(l *lua.LState) int {
	id := strings.TrimSpace(l.CheckString(1))
	spec := copyLuaTable(l, l.CheckTable(2))
	if id == "" {
		l.RaiseError("job id is required")
		return 0
	}
	spec.RawSetString("id", lua.LString(id))
	l.Push(spec)
	return 1
}

func (v *VM) luaStringOption(l *lua.LState) int      { return v.luaOption(l, "string") }
func (v *VM) luaBoolOption(l *lua.LState) int        { return v.luaOption(l, "bool") }
func (v *VM) luaIntOption(l *lua.LState) int         { return v.luaOption(l, "int") }
func (v *VM) luaFloatOption(l *lua.LState) int       { return v.luaOption(l, "float") }
func (v *VM) luaUserOption(l *lua.LState) int        { return v.luaOption(l, "user") }
func (v *VM) luaChannelOption(l *lua.LState) int     { return v.luaOption(l, "channel") }
func (v *VM) luaRoleOption(l *lua.LState) int        { return v.luaOption(l, "role") }
func (v *VM) luaMentionableOption(l *lua.LState) int { return v.luaOption(l, "mentionable") }
func (v *VM) luaAttachmentOption(l *lua.LState) int  { return v.luaOption(l, "attachment") }

func (v *VM) luaOption(l *lua.LState, typ string) int {
	name := strings.TrimSpace(l.CheckString(1))
	spec := copyLuaTable(l, l.CheckTable(2))
	if name == "" {
		l.RaiseError("option name is required")
		return 0
	}
	spec.RawSetString("name", lua.LString(name))
	spec.RawSetString("type", lua.LString(typ))
	l.Push(spec)
	return 1
}

func (v *VM) luaReply(l *lua.LState) int {
	spec := copyLuaTable(l, l.CheckTable(1))
	spec.RawSetString("type", lua.LString("message"))
	l.Push(spec)
	return 1
}

func (v *VM) luaUpdate(l *lua.LState) int {
	spec := copyLuaTable(l, l.CheckTable(1))
	spec.RawSetString("type", lua.LString("update"))
	l.Push(spec)
	return 1
}

func (v *VM) luaModal(l *lua.LState) int {
	id := strings.TrimSpace(l.CheckString(1))
	spec := copyLuaTable(l, l.CheckTable(2))
	if id == "" {
		l.RaiseError("modal id is required")
		return 0
	}
	spec.RawSetString("type", lua.LString("modal"))
	spec.RawSetString("id", lua.LString(id))
	l.Push(spec)
	return 1
}

func (v *VM) luaPresent(l *lua.LState) int {
	spec := copyLuaTable(l, l.CheckTable(1))
	present := l.NewTable()
	for _, key := range []string{"kind", "title", "body", "fields"} {
		value := spec.RawGetString(key)
		if value != lua.LNil {
			present.RawSetString(key, value)
		}
	}

	out := l.NewTable()
	out.RawSetString("present", present)
	for _, key := range []string{"ephemeral", "components", "content", "embeds"} {
		value := spec.RawGetString(key)
		if value != lua.LNil {
			out.RawSetString(key, value)
		}
	}
	l.Push(out)
	return 1
}

func (v *VM) luaButton(l *lua.LState) int {
	id := strings.TrimSpace(l.CheckString(1))
	spec := copyLuaTable(l, l.CheckTable(2))
	if id == "" {
		l.RaiseError("button id is required")
		return 0
	}
	spec.RawSetString("type", lua.LString("button"))
	spec.RawSetString("id", lua.LString(id))
	l.Push(spec)
	return 1
}

func (v *VM) luaTextInput(l *lua.LState) int {
	id := strings.TrimSpace(l.CheckString(1))
	spec := copyLuaTable(l, l.CheckTable(2))
	if id == "" {
		l.RaiseError("text input id is required")
		return 0
	}
	spec.RawSetString("id", lua.LString(id))
	l.Push(spec)
	return 1
}

func (v *VM) luaRandomInt(l *lua.LState) int {
	min := l.CheckInt(1)
	max := l.CheckInt(2)
	if max < min {
		l.RaiseError("max must be >= min")
		return 0
	}

	n, err := cryptoRandIntInclusive(min, max)
	if err != nil {
		l.RaiseError("random failed")
		return 0
	}
	l.Push(lua.LNumber(n))
	return 1
}

func (v *VM) luaRandomChoice(l *lua.LState) int {
	list := l.CheckTable(1)
	length := list.Len()
	if length == 0 {
		l.RaiseError("choice requires a non-empty list")
		return 0
	}

	index, err := cryptoRandIntInclusive(1, length)
	if err != nil {
		l.RaiseError("random failed")
		return 0
	}
	l.Push(list.RawGetInt(index))
	return 1
}

func (v *VM) luaKawaiiGIF(l *lua.LState) int {
	if !v.perms.Integrations.Kawaii {
		l.RaiseError("permission denied: integrations.kawaii")
		return 0
	}
	if v.kawaii == nil {
		l.RaiseError("kawaii unavailable")
		return 0
	}

	rawEndpoint := strings.ToLower(strings.TrimSpace(l.CheckString(1)))
	var endpoint kawaii.Endpoint
	switch rawEndpoint {
	case string(kawaii.EndpointHug):
		endpoint = kawaii.EndpointHug
	case string(kawaii.EndpointPat):
		endpoint = kawaii.EndpointPat
	case string(kawaii.EndpointPoke):
		endpoint = kawaii.EndpointPoke
	case string(kawaii.EndpointShrug):
		endpoint = kawaii.EndpointShrug
	default:
		l.RaiseError("unsupported kawaii endpoint")
		return 0
	}

	gifURL, err := v.kawaii.FetchGIF(v.ctx(), endpoint)
	if err != nil {
		l.RaiseError("kawaii fetch failed")
		return 0
	}
	l.Push(lua.LString(gifURL))
	return 1
}

func (v *VM) luaEffectSendChannel(l *lua.LState) int {
	spec := copyLuaTable(l, l.CheckTable(1))
	action := l.NewTable()
	action.RawSetString("type", lua.LString("send_channel"))
	if channelID := spec.RawGetString("channel_id"); channelID != lua.LNil {
		action.RawSetString("channel_id", channelID)
	}
	action.RawSetString("message", spec.RawGetString("message"))

	actions := l.NewTable()
	actions.RawSetInt(1, action)
	out := l.NewTable()
	out.RawSetString("actions", actions)
	l.Push(out)
	return 1
}

func (v *VM) luaEffectSendDM(l *lua.LState) int {
	spec := copyLuaTable(l, l.CheckTable(1))
	action := l.NewTable()
	action.RawSetString("type", lua.LString("send_dm"))
	if userID := spec.RawGetString("user_id"); userID != lua.LNil {
		action.RawSetString("user_id", userID)
	}
	action.RawSetString("message", spec.RawGetString("message"))

	actions := l.NewTable()
	actions.RawSetInt(1, action)
	out := l.NewTable()
	out.RawSetString("actions", actions)
	l.Push(out)
	return 1
}

func (v *VM) luaRequire(l *lua.LState) int {
	relPath := strings.TrimSpace(l.CheckString(1))
	if relPath == "" {
		l.RaiseError("module path is required")
		return 0
	}

	cleanPath, absPath, err := v.resolveLocalLuaPath(relPath)
	if err != nil {
		l.RaiseError("%s", err.Error())
		return 0
	}

	if cached, ok := v.moduleCache[cleanPath]; ok {
		l.Push(cached)
		return 1
	}

	fn, loadErr := l.LoadFile(absPath)
	if loadErr != nil {
		l.RaiseError("module load failed")
		return 0
	}

	l.Push(fn)
	if callErr := l.PCall(0, 1, nil); callErr != nil {
		l.RaiseError("module load failed")
		return 0
	}

	module := l.Get(-1)
	l.Pop(1)
	if module == lua.LNil {
		module = lua.LTrue
	}

	v.moduleCache[cleanPath] = module
	l.Push(module)
	return 1
}

func (v *VM) resolveLocalLuaPath(rel string) (string, string, error) {
	if strings.Contains(rel, "\\") {
		return "", "", fmt.Errorf("invalid lua path")
	}
	if !strings.HasSuffix(strings.ToLower(rel), ".lua") {
		return "", "", fmt.Errorf("lua path must end with .lua")
	}
	if strings.HasPrefix(rel, "/") {
		return "", "", fmt.Errorf("lua path must be relative")
	}

	clean := filepath.Clean(rel)
	if clean == "." || strings.HasPrefix(clean, "..") || strings.Contains(clean, "/..") {
		return "", "", fmt.Errorf("lua path escapes plugin dir")
	}

	baseAbs, err := filepath.Abs(v.dir)
	if err != nil {
		return "", "", fmt.Errorf("lua path error")
	}
	targetAbs, err := filepath.Abs(filepath.Join(v.dir, clean))
	if err != nil {
		return "", "", fmt.Errorf("lua path error")
	}

	relToBase, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return "", "", fmt.Errorf("lua path escapes plugin dir")
	}
	relToBaseSlash := filepath.ToSlash(relToBase)
	if relToBase == "." || strings.HasPrefix(relToBase, "..") || strings.HasPrefix(relToBaseSlash, "../") {
		return "", "", fmt.Errorf("lua path escapes plugin dir")
	}

	fi, err := os.Stat(targetAbs)
	if err != nil {
		return "", "", fmt.Errorf("lua file not found")
	}
	if fi.Size() > 128*1024 {
		return "", "", fmt.Errorf("lua file too large")
	}

	return filepath.ToSlash(clean), targetAbs, nil
}

func copyLuaTable(l *lua.LState, source *lua.LTable) *lua.LTable {
	clone := l.NewTable()
	source.ForEach(func(key, value lua.LValue) {
		clone.RawSet(key, value)
	})
	return clone
}

func cryptoRandIntInclusive(min, max int) (int, error) {
	if max < min {
		return 0, fmt.Errorf("invalid range")
	}
	width := max - min + 1
	if width <= 0 {
		return 0, fmt.Errorf("invalid range")
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(width)))
	if err != nil {
		return 0, err
	}
	return min + int(n.Int64()), nil
}

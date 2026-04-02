package luaplugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	lua "github.com/yuin/gopher-lua"

	"github.com/xsyetopz/go-mamusiabtw/internal/i18n"
	"github.com/xsyetopz/go-mamusiabtw/internal/permissions"
	"github.com/xsyetopz/go-mamusiabtw/internal/store"
)

const defaultCallTimeout = 2 * time.Second

type Options struct {
	Logger *slog.Logger

	PluginID    string
	PluginDir   string
	Permissions permissions.Permissions

	Store store.PluginKVStore
	I18n  *i18n.Registry
}

type Payload struct {
	GuildID   string
	ChannelID string
	UserID    string
	Locale    string
	Options   map[string]any
}

type VM struct {
	mu sync.Mutex

	logger *slog.Logger
	plugin string
	dir    string
	perms  permissions.Permissions
	store  store.PluginKVStore
	i18n   *i18n.Registry

	L *lua.LState

	definition  *pluginDefinition
	moduleCache map[string]lua.LValue

	execCtx context.Context
	locale  string
}

func NewFromFile(fileName string, opts Options) (*VM, error) {
	if strings.TrimSpace(fileName) == "" {
		return nil, errors.New("lua filename is required")
	}
	if strings.TrimSpace(opts.PluginID) == "" {
		return nil, errors.New("plugin id is required")
	}
	if strings.TrimSpace(opts.PluginDir) == "" {
		return nil, errors.New("plugin dir is required")
	}
	if opts.Logger == nil {
		return nil, errors.New("logger is required")
	}

	l := lua.NewState(lua.Options{
		SkipOpenLibs: true,
	})

	if err := openSafeLibs(l); err != nil {
		l.Close()
		return nil, err
	}
	stripDangerousGlobals(l)

	vm := &VM{
		logger:      opts.Logger.With(slog.String("component", "lua")),
		plugin:      opts.PluginID,
		dir:         opts.PluginDir,
		perms:       opts.Permissions,
		store:       opts.Store,
		i18n:        opts.I18n,
		L:           l,
		moduleCache: map[string]lua.LValue{},
	}

	vm.registerHostAPI()

	abs, err := filepath.Abs(fileName)
	if err != nil {
		l.Close()
		return nil, fmt.Errorf("abs path: %w", err)
	}

	definition, err := vm.loadEntryFile(abs)
	if err != nil {
		l.Close()
		return nil, fmt.Errorf("load lua plugin %q: %w", fileName, err)
	}
	vm.definition = definition

	return vm, nil
}

func (v *VM) Close() {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.L == nil {
		return
	}
	v.L.Close()
	v.L = nil
}

func (v *VM) HasFunc(funcName string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.L == nil {
		return false
	}
	fn := v.L.GetGlobal(funcName)
	return fn.Type() == lua.LTFunction
}

// CallHandle calls a Lua function by name with (cmd, ctxTable) and returns its result as a Go value.
//
// Allowed return types:
// - nil
// - string
// - bool
// - float64
// - []any / map[string]any (for tables; depth-limited).
func (v *VM) CallHandle(ctx context.Context, funcName string, cmd string, payload Payload) (any, bool, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.L == nil {
		return nil, false, errors.New("lua vm is closed")
	}

	fn := v.L.GetGlobal(funcName)
	if fn.Type() != lua.LTFunction {
		return nil, false, fmt.Errorf("lua function %q not found", funcName)
	}

	timeoutCtx := ctx
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		timeoutCtx, cancel = context.WithTimeout(ctx, defaultCallTimeout)
		defer cancel()
	}

	prevCtx := v.L.RemoveContext()
	v.L.SetContext(timeoutCtx)
	defer func() {
		_ = v.L.RemoveContext()
		if prevCtx != nil {
			v.L.SetContext(prevCtx)
		}
	}()

	v.execCtx = timeoutCtx
	v.locale = strings.TrimSpace(payload.Locale)
	defer func() {
		v.execCtx = nil
		v.locale = ""
	}()

	payloadTable, err := v.payloadToLua(payload)
	if err != nil {
		return nil, false, err
	}

	v.L.Push(fn)
	v.L.Push(lua.LString(cmd))
	v.L.Push(payloadTable)

	const (
		pcallNArgs = 2
		pcallNRet  = 1
	)
	if callErr := v.L.PCall(pcallNArgs, pcallNRet, nil); callErr != nil {
		return nil, false, fmt.Errorf("lua call %q: %w", funcName, callErr)
	}

	res := v.L.Get(-1)
	v.L.Pop(1)

	if res == lua.LNil {
		return nil, false, nil
	}

	out, _, err := luaToAny(res)
	if err != nil {
		return nil, false, fmt.Errorf("lua %q return: %w", funcName, err)
	}
	return out, true, nil
}

func openSafeLibs(l *lua.LState) error {
	// Intentionally do not open: os, io, package, debug.
	for _, pair := range []struct {
		n string
		f lua.LGFunction
	}{
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
	} {
		if err := l.CallByParam(lua.P{
			Fn:      l.NewFunction(pair.f),
			NRet:    0,
			Protect: true,
		}, lua.LString(pair.n)); err != nil {
			return fmt.Errorf("open lua lib %q: %w", pair.n, err)
		}
	}
	return nil
}

func stripDangerousGlobals(l *lua.LState) {
	// Base lib in Lua includes file-loading helpers; remove them.
	l.SetGlobal("dofile", lua.LNil)
	l.SetGlobal("loadfile", lua.LNil)
	l.SetGlobal("require", lua.LNil)
	l.SetGlobal("module", lua.LNil)

	// Sandboxing hardening: prevent plugins from mutating their global environment via setfenv (Lua 5.1).
	l.SetGlobal("setfenv", lua.LNil)
	l.SetGlobal("getfenv", lua.LNil)
	l.SetGlobal("load", lua.LNil)
	l.SetGlobal("loadstring", lua.LNil)
}

func (v *VM) registerHostAPI() {
	bot := v.L.NewTable()
	logTable := v.L.NewTable()
	i18nTable := v.L.NewTable()
	storeTable := v.L.NewTable()
	optionTable := v.L.NewTable()
	uiTable := v.L.NewTable()
	effectsTable := v.L.NewTable()

	logTable.RawSetString("info", v.L.NewFunction(v.luaLog))

	i18nTable.RawSetString("t", v.L.NewFunction(v.luaT))

	storeTable.RawSetString("get", v.L.NewFunction(v.luaKVGet))
	storeTable.RawSetString("put", v.L.NewFunction(v.luaKVPut))
	storeTable.RawSetString("del", v.L.NewFunction(v.luaKVDel))
	storeTable.RawSetString("get_json", v.L.NewFunction(v.luaKVGetJSON))
	storeTable.RawSetString("put_json", v.L.NewFunction(v.luaKVPutJSON))

	optionTable.RawSetString("string", v.L.NewFunction(v.luaStringOption))
	optionTable.RawSetString("bool", v.L.NewFunction(v.luaBoolOption))
	optionTable.RawSetString("int", v.L.NewFunction(v.luaIntOption))
	optionTable.RawSetString("float", v.L.NewFunction(v.luaFloatOption))
	optionTable.RawSetString("user", v.L.NewFunction(v.luaUserOption))
	optionTable.RawSetString("channel", v.L.NewFunction(v.luaChannelOption))
	optionTable.RawSetString("role", v.L.NewFunction(v.luaRoleOption))
	optionTable.RawSetString("mentionable", v.L.NewFunction(v.luaMentionableOption))
	optionTable.RawSetString("attachment", v.L.NewFunction(v.luaAttachmentOption))

	uiTable.RawSetString("reply", v.L.NewFunction(v.luaReply))
	uiTable.RawSetString("update", v.L.NewFunction(v.luaUpdate))
	uiTable.RawSetString("modal", v.L.NewFunction(v.luaModal))
	uiTable.RawSetString("present", v.L.NewFunction(v.luaPresent))
	uiTable.RawSetString("button", v.L.NewFunction(v.luaButton))
	uiTable.RawSetString("text_input", v.L.NewFunction(v.luaTextInput))

	effectsTable.RawSetString("send_channel", v.L.NewFunction(v.luaEffectSendChannel))
	effectsTable.RawSetString("send_dm", v.L.NewFunction(v.luaEffectSendDM))

	bot.RawSetString("log", logTable)
	bot.RawSetString("i18n", i18nTable)
	bot.RawSetString("store", storeTable)
	bot.RawSetString("option", optionTable)
	bot.RawSetString("ui", uiTable)
	bot.RawSetString("effects", effectsTable)
	bot.RawSetString("plugin", v.L.NewFunction(v.luaPlugin))
	bot.RawSetString("command", v.L.NewFunction(v.luaCommand))
	bot.RawSetString("job", v.L.NewFunction(v.luaJob))
	bot.RawSetString("require", v.L.NewFunction(v.luaRequire))
	bot.RawSetString("include", v.L.NewFunction(v.luaInclude))

	legacy := v.L.NewTable()
	legacy.RawSetString("log", logTable.RawGetString("info"))
	legacy.RawSetString("include", bot.RawGetString("include"))
	legacy.RawSetString("t", i18nTable.RawGetString("t"))
	legacy.RawSetString("kv_get", storeTable.RawGetString("get"))
	legacy.RawSetString("kv_put", storeTable.RawGetString("put"))
	legacy.RawSetString("kv_del", storeTable.RawGetString("del"))
	legacy.RawSetString("kv_get_json", storeTable.RawGetString("get_json"))
	legacy.RawSetString("kv_put_json", storeTable.RawGetString("put_json"))

	v.L.SetGlobal("bot", bot)
	v.L.SetGlobal("mamusiabtw", legacy)
}

func (v *VM) payloadToLua(p Payload) (*lua.LTable, error) {
	t := v.L.NewTable()
	t.RawSetString("guild_id", lua.LString(strings.TrimSpace(p.GuildID)))
	t.RawSetString("channel_id", lua.LString(strings.TrimSpace(p.ChannelID)))
	t.RawSetString("user_id", lua.LString(strings.TrimSpace(p.UserID)))
	t.RawSetString("locale", lua.LString(strings.TrimSpace(p.Locale)))

	opts := v.L.NewTable()
	for name, val := range p.Options {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		lv, err := anyToLuaValue(v.L, val, 0)
		if err != nil {
			return nil, fmt.Errorf("option %q: %w", name, err)
		}
		opts.RawSetString(name, lv)
	}
	t.RawSetString("options", opts)

	return t, nil
}

func (v *VM) luaLog(l *lua.LState) int {
	msg := strings.TrimSpace(l.CheckString(1))
	if msg == "" {
		return 0
	}
	v.logger.Info("plugin log", slog.String("plugin", v.plugin), slog.String("msg", msg))
	return 0
}

func (v *VM) luaInclude(l *lua.LState) int {
	rel := strings.TrimSpace(l.CheckString(1))
	if rel == "" {
		l.RaiseError("include path is required")
		return 0
	}
	if strings.Contains(rel, "\\") {
		l.RaiseError("invalid include path")
		return 0
	}
	if !strings.HasSuffix(strings.ToLower(rel), ".lua") {
		l.RaiseError("include path must end with .lua")
		return 0
	}
	if strings.HasPrefix(rel, "/") {
		l.RaiseError("include path must be relative")
		return 0
	}

	clean := filepath.Clean(rel)
	if clean == "." || strings.HasPrefix(clean, "..") || strings.Contains(clean, "/..") {
		l.RaiseError("invalid include path")
		return 0
	}

	baseAbs, err := filepath.Abs(v.dir)
	if err != nil {
		l.RaiseError("include error")
		return 0
	}
	targetAbs, err := filepath.Abs(filepath.Join(v.dir, clean))
	if err != nil {
		l.RaiseError("include error")
		return 0
	}

	// Ensure targetAbs stays within baseAbs.
	relToBase, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		l.RaiseError("include path escapes plugin dir")
		return 0
	}
	relToBaseSlash := filepath.ToSlash(relToBase)
	if relToBase == "." || strings.HasPrefix(relToBase, "..") || strings.HasPrefix(relToBaseSlash, "../") {
		l.RaiseError("include path escapes plugin dir")
		return 0
	}

	fi, err := os.Stat(targetAbs)
	if err != nil {
		l.RaiseError("include file not found")
		return 0
	}
	if fi.Size() > 128*1024 {
		l.RaiseError("include file too large")
		return 0
	}

	if doErr := l.DoFile(targetAbs); doErr != nil {
		l.RaiseError("include failed")
		return 0
	}

	l.Push(lua.LTrue)
	return 1
}

func (v *VM) luaT(l *lua.LState) int {
	const (
		argID           = 1
		argTemplateData = 2
		argPlural       = 3
	)

	id := strings.TrimSpace(l.CheckString(argID))
	if id == "" {
		l.Push(lua.LString(""))
		return 1
	}

	data, ok := luaOptTemplateData(l, argTemplateData)
	if !ok {
		return 0
	}

	plural, ok := luaOptAny(l, argPlural, "invalid plural count")
	if !ok {
		return 0
	}

	if v.i18n == nil {
		l.Push(lua.LString(id))
		return 1
	}

	s := v.i18n.MustLocalize(i18n.Config{
		Locale:       v.locale,
		PluginID:     v.plugin,
		MessageID:    id,
		TemplateData: data,
		PluralCount:  plural,
	})
	l.Push(lua.LString(s))
	return 1
}

func luaOptTemplateData(l *lua.LState, idx int) (map[string]any, bool) {
	if l.GetTop() < idx {
		return nil, true
	}
	raw := l.Get(idx)
	if raw == lua.LNil {
		return nil, true
	}

	val, _, err := luaToAny(raw)
	if err != nil {
		l.RaiseError("invalid template data")
		return nil, false
	}
	m, ok := val.(map[string]any)
	if !ok {
		l.RaiseError("template data must be an object")
		return nil, false
	}
	return m, true
}

func luaOptAny(l *lua.LState, idx int, errMsg string) (any, bool) {
	if l.GetTop() < idx {
		return nil, true
	}
	raw := l.Get(idx)
	if raw == lua.LNil {
		return nil, true
	}

	val, _, err := luaToAny(raw)
	if err != nil {
		l.RaiseError("%s", errMsg)
		return nil, false
	}
	return val, true
}

func (v *VM) luaKVGet(l *lua.LState) int {
	const (
		argGuildID = 1
		argKey     = 2
		retPair    = 2
	)

	if !v.perms.Storage.KV {
		l.RaiseError("permission denied: storage.kv")
		return 0
	}
	if v.store == nil {
		l.RaiseError("storage unavailable")
		return 0
	}

	guildID := parseSnowflakeString(l.CheckString(argGuildID))
	key := strings.TrimSpace(l.CheckString(argKey))
	if guildID == 0 || key == "" {
		l.Push(lua.LNil)
		l.Push(lua.LFalse)
		return retPair
	}

	value, ok, err := v.store.GetPluginKV(v.ctx(), guildID, v.plugin, key)
	if err != nil {
		l.RaiseError("storage error")
		return 0
	}

	if !ok {
		l.Push(lua.LNil)
		l.Push(lua.LFalse)
		return retPair
	}

	var decoded any
	if unmarshalErr := json.Unmarshal([]byte(value), &decoded); unmarshalErr != nil {
		l.RaiseError("storage decode error")
		return 0
	}
	lv, err := anyToLuaValue(l, decoded, 0)
	if err != nil {
		l.RaiseError("storage decode error")
		return 0
	}

	l.Push(lv)
	l.Push(lua.LTrue)
	return retPair
}

func (v *VM) luaKVPut(l *lua.LState) int {
	const (
		argGuildID = 1
		argKey     = 2
		argValue   = 3
	)

	if !v.perms.Storage.KV {
		l.RaiseError("permission denied: storage.kv")
		return 0
	}
	if v.store == nil {
		l.RaiseError("storage unavailable")
		return 0
	}

	guildID := parseSnowflakeString(l.CheckString(argGuildID))
	key := strings.TrimSpace(l.CheckString(argKey))
	value := l.CheckAny(argValue)

	if guildID == 0 || key == "" {
		l.RaiseError("invalid guild_id or key")
		return 0
	}

	goVal, _, err := luaToAny(value)
	if err != nil {
		l.RaiseError("invalid value")
		return 0
	}
	enc, err := json.Marshal(goVal)
	if err != nil {
		l.RaiseError("value must be JSON encodable")
		return 0
	}
	if len(enc) > 16*1024 {
		l.RaiseError("value too large")
		return 0
	}

	if putErr := v.store.PutPluginKV(v.ctx(), guildID, v.plugin, key, string(enc)); putErr != nil {
		l.RaiseError("storage error")
		return 0
	}

	l.Push(lua.LTrue)
	return 1
}

func (v *VM) luaKVDel(l *lua.LState) int {
	const (
		argGuildID = 1
		argKey     = 2
	)

	if !v.perms.Storage.KV {
		l.RaiseError("permission denied: storage.kv")
		return 0
	}
	if v.store == nil {
		l.RaiseError("storage unavailable")
		return 0
	}

	guildID := parseSnowflakeString(l.CheckString(argGuildID))
	key := strings.TrimSpace(l.CheckString(argKey))
	if guildID == 0 || key == "" {
		l.RaiseError("invalid guild_id or key")
		return 0
	}

	if err := v.store.DeletePluginKV(v.ctx(), guildID, v.plugin, key); err != nil {
		l.RaiseError("storage error")
		return 0
	}
	l.Push(lua.LTrue)
	return 1
}

func (v *VM) luaKVGetJSON(l *lua.LState) int {
	const (
		argGuildID = 1
		argKey     = 2
		retPair    = 2
	)

	if !v.perms.Storage.KV {
		l.RaiseError("permission denied: storage.kv")
		return 0
	}
	if v.store == nil {
		l.RaiseError("storage unavailable")
		return 0
	}

	guildID := parseSnowflakeString(l.CheckString(argGuildID))
	key := strings.TrimSpace(l.CheckString(argKey))
	if guildID == 0 || key == "" {
		l.Push(lua.LNil)
		l.Push(lua.LFalse)
		return retPair
	}

	value, ok, err := v.store.GetPluginKV(v.ctx(), guildID, v.plugin, key)
	if err != nil {
		l.RaiseError("storage error")
		return 0
	}

	if !ok {
		l.Push(lua.LNil)
		l.Push(lua.LFalse)
		return retPair
	}
	l.Push(lua.LString(value))
	l.Push(lua.LTrue)
	return retPair
}

func (v *VM) luaKVPutJSON(l *lua.LState) int {
	const (
		argGuildID = 1
		argKey     = 2
		argValue   = 3
	)

	if !v.perms.Storage.KV {
		l.RaiseError("permission denied: storage.kv")
		return 0
	}
	if v.store == nil {
		l.RaiseError("storage unavailable")
		return 0
	}

	guildID := parseSnowflakeString(l.CheckString(argGuildID))
	key := strings.TrimSpace(l.CheckString(argKey))
	value := l.CheckString(argValue)

	if guildID == 0 || key == "" {
		l.RaiseError("invalid guild_id or key")
		return 0
	}
	if !json.Valid([]byte(value)) {
		l.RaiseError("value must be JSON")
		return 0
	}
	if len(value) > 16*1024 {
		l.RaiseError("value too large")
		return 0
	}

	if putErr := v.store.PutPluginKV(v.ctx(), guildID, v.plugin, key, value); putErr != nil {
		l.RaiseError("storage error")
		return 0
	}

	l.Push(lua.LTrue)
	return 1
}

func (v *VM) ctx() context.Context {
	if v.execCtx != nil {
		return v.execCtx
	}
	return context.Background()
}

func parseSnowflakeString(raw string) uint64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

const maxTableDepth = 16
const maxTableItems = 500

func luaToAny(v lua.LValue) (any, bool, error) {
	return luaToAnyValue(v, 0)
}

func tableToAny(t *lua.LTable, depth int) (any, error) {
	if depth > maxTableDepth {
		return nil, errors.New("too deep")
	}
	kind, maxIndex, err := tableKind(t)
	if err != nil {
		return nil, err
	}
	if kind == tableKindArray {
		return tableToArray(t, maxIndex, depth)
	}
	return tableToObject(t, depth)
}

type luaTableKind int

const (
	tableKindArray luaTableKind = iota
	tableKindObject
)

func luaToAnyValue(v lua.LValue, depth int) (any, bool, error) {
	if depth > maxTableDepth {
		return nil, false, errors.New("too deep")
	}

	switch vv := v.(type) {
	case *lua.LNilType:
		return nil, true, nil
	case lua.LString:
		return string(vv), true, nil
	case lua.LBool:
		return bool(vv), true, nil
	case lua.LNumber:
		return float64(vv), true, nil
	case *lua.LTable:
		out, err := tableToAny(vv, depth+1)
		if err != nil {
			return nil, false, err
		}
		return out, true, nil
	default:
		return nil, false, fmt.Errorf("unsupported lua type %s", v.Type().String())
	}
}

func tableKind(t *lua.LTable) (luaTableKind, int, error) {
	seen := 0
	maxIndex := 0
	hasIntKeys := false
	hasStringKeys := false

	t.ForEach(func(k, _ lua.LValue) {
		if seen > maxTableItems {
			return
		}
		seen++
		switch kk := k.(type) {
		case lua.LNumber:
			if float64(kk) == float64(int(kk)) && int(kk) >= 1 {
				hasIntKeys = true
				if int(kk) > maxIndex {
					maxIndex = int(kk)
				}
			} else {
				hasStringKeys = true
			}
		case lua.LString:
			hasStringKeys = true
		default:
			hasStringKeys = true
		}
	})

	if seen > maxTableItems {
		return tableKindObject, 0, errors.New("too many items")
	}
	if hasIntKeys && hasStringKeys {
		return tableKindObject, 0, errors.New("mixed keys")
	}
	if hasIntKeys {
		if maxIndex > maxTableItems {
			return tableKindObject, 0, errors.New("too many items")
		}
		return tableKindArray, maxIndex, nil
	}
	return tableKindObject, 0, nil
}

func tableToArray(t *lua.LTable, maxIndex int, depth int) ([]any, error) {
	out := make([]any, maxIndex)
	for i := 1; i <= maxIndex; i++ {
		lv := t.RawGetInt(i)
		vv, _, err := luaToAnyValue(lv, depth+1)
		if err != nil {
			return nil, err
		}
		out[i-1] = vv
	}
	return out, nil
}

func tableToObject(t *lua.LTable, depth int) (map[string]any, error) {
	out := map[string]any{}
	var firstErr error
	t.ForEach(func(k, v lua.LValue) {
		if firstErr != nil {
			return
		}
		if len(out) >= maxTableItems {
			firstErr = errors.New("too many items")
			return
		}

		ks, ok := k.(lua.LString)
		if !ok {
			firstErr = errors.New("object key must be a string")
			return
		}
		key := strings.TrimSpace(string(ks))
		if key == "" {
			firstErr = errors.New("object key cannot be empty")
			return
		}

		vv, _, err := luaToAnyValue(v, depth+1)
		if err != nil {
			firstErr = err
			return
		}
		out[key] = vv
	})
	if firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

func anyToLuaValue(l *lua.LState, v any, depth int) (lua.LValue, error) {
	if depth > maxTableDepth {
		return lua.LNil, errors.New("too deep")
	}

	if lv, ok := anyToLuaScalar(v); ok {
		return lv, nil
	}
	return anyToLuaComposite(l, v, depth)
}

func anyToLuaScalar(v any) (lua.LValue, bool) {
	switch vv := v.(type) {
	case nil:
		return lua.LNil, true
	case string:
		return lua.LString(vv), true
	case bool:
		return lua.LBool(vv), true
	case int:
		return lua.LNumber(vv), true
	case int64:
		return lua.LNumber(vv), true
	case uint64:
		// Snowflakes do not fit safely into Lua numbers (float64); keep as string.
		return lua.LString(strconv.FormatUint(vv, 10)), true
	case float64:
		return lua.LNumber(vv), true
	case float32:
		return lua.LNumber(vv), true
	default:
		return lua.LNil, false
	}
}

func anyToLuaComposite(l *lua.LState, v any, depth int) (lua.LValue, error) {
	switch vv := v.(type) {
	case []string:
		return anyToLuaSlice(l, stringSliceToAny(vv), depth)
	case map[string]string:
		return anyToLuaMap(l, stringMapToAny(vv), depth)
	case []any:
		return anyToLuaSlice(l, vv, depth)
	case map[string]any:
		return anyToLuaMap(l, vv, depth)
	default:
		return lua.LNil, fmt.Errorf("unsupported type %T", v)
	}
}

func anyToLuaSlice(l *lua.LState, vv []any, depth int) (lua.LValue, error) {
	if len(vv) > maxTableItems {
		return lua.LNil, errors.New("too many items")
	}

	t := l.NewTable()
	for i, item := range vv {
		lv, err := anyToLuaValue(l, item, depth+1)
		if err != nil {
			return lua.LNil, err
		}
		t.RawSetInt(i+1, lv)
	}
	return t, nil
}

func anyToLuaMap(l *lua.LState, vv map[string]any, depth int) (lua.LValue, error) {
	if len(vv) > maxTableItems {
		return lua.LNil, errors.New("too many items")
	}

	t := l.NewTable()
	for k, item := range vv {
		if strings.TrimSpace(k) == "" {
			continue
		}
		lv, err := anyToLuaValue(l, item, depth+1)
		if err != nil {
			return lua.LNil, err
		}
		t.RawSetString(k, lv)
	}
	return t, nil
}

func stringSliceToAny(vv []string) []any {
	out := make([]any, 0, len(vv))
	for _, s := range vv {
		out = append(out, s)
	}
	return out
}

func stringMapToAny(vv map[string]string) map[string]any {
	out := make(map[string]any, len(vv))
	for k, s := range vv {
		out[k] = s
	}
	return out
}

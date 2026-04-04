package luaplugin

import (
	"strings"

	"github.com/xsyetopz/go-mamusiabtw/internal/buildinfo"
	lua "github.com/yuin/gopher-lua"
)

func (v *VM) luaRuntimeBuildInfo(l *lua.LState) int {
	info := map[string]any{
		"version":            strings.TrimSpace(buildinfo.Version),
		"description":        strings.TrimSpace(buildinfo.Description),
		"repository":         strings.TrimSpace(buildinfo.Repository),
		"mascot_image_url":   strings.TrimSpace(buildinfo.MascotImageURL),
		"developer_url":      strings.TrimSpace(buildinfo.DeveloperURL),
		"support_server_url": strings.TrimSpace(buildinfo.SupportServerURL),
	}

	value, err := anyToLuaValue(l, info, 0)
	if err != nil {
		l.RaiseError("runtime build info unavailable")
		return 0
	}
	l.Push(value)
	return 1
}

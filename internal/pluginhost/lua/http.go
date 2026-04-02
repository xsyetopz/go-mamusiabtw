package luaplugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

const (
	defaultPluginHTTPTimeout  = 3 * time.Second
	defaultPluginHTTPMaxBytes = 64 * 1024
	maxPluginHTTPMaxBytes     = 1024 * 1024
)

func pluginHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: defaultPluginHTTPTimeout}
}

func (v *VM) luaHTTPGet(l *lua.LState) int {
	resp, err := v.performHTTPGet(l)
	if err != nil {
		l.RaiseError("%s", err.Error())
		return 0
	}

	body, readErr := v.readHTTPBody(resp)
	if readErr != nil {
		_ = resp.Body.Close()
		l.RaiseError("%s", readErr.Error())
		return 0
	}
	defer resp.Body.Close()

	out := l.NewTable()
	out.RawSetString("status", lua.LNumber(resp.StatusCode))
	out.RawSetString("body", lua.LString(body))
	out.RawSetString("headers", luaHeaderTable(l, resp.Header))
	l.Push(out)
	return 1
}

func (v *VM) luaHTTPGetJSON(l *lua.LState) int {
	resp, err := v.performHTTPGet(l)
	if err != nil {
		l.RaiseError("%s", err.Error())
		return 0
	}

	body, readErr := v.readHTTPBody(resp)
	if readErr != nil {
		_ = resp.Body.Close()
		l.RaiseError("%s", readErr.Error())
		return 0
	}
	defer resp.Body.Close()

	var decoded any
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		l.RaiseError("http json decode failed")
		return 0
	}

	lv, err := anyToLuaValue(l, decoded, 0)
	if err != nil {
		l.RaiseError("http json decode failed")
		return 0
	}
	l.Push(lv)
	return 1
}

func (v *VM) performHTTPGet(l *lua.LState) (*http.Response, error) {
	if !v.perms.Network.HTTP {
		return nil, errors.New("permission denied: network.http")
	}
	if v.http == nil {
		return nil, errors.New("http unavailable")
	}

	spec := l.CheckTable(1)
	rawURL := strings.TrimSpace(luaTableString(spec, "url"))
	if rawURL == "" {
		return nil, errors.New("http url is required")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.New("http url is invalid")
	}
	if strings.ToLower(parsed.Scheme) != "https" {
		return nil, errors.New("http url must be https")
	}
	if strings.TrimSpace(parsed.Hostname()) == "" {
		return nil, errors.New("http url must include a host")
	}

	req, err := http.NewRequestWithContext(v.ctx(), http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, errors.New("http request failed")
	}

	maxBytes, err := luaTableMaxBytes(spec)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(context.WithValue(req.Context(), httpMaxBytesContextKey{}, maxBytes))
	req.Header.Set("Accept", "application/json")

	headers, err := luaHeaders(spec.RawGetString("headers"))
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := v.http.Do(req)
	if err != nil {
		return nil, errors.New("http request failed")
	}
	return resp, nil
}

func (v *VM) readHTTPBody(resp *http.Response) (string, error) {
	limit := defaultPluginHTTPMaxBytes
	if resp == nil || resp.Request == nil {
		return "", errors.New("http response missing")
	}

	if reqLimit := resp.Request.Context().Value(httpMaxBytesContextKey{}); reqLimit != nil {
		if maxBytes, ok := reqLimit.(int); ok && maxBytes > 0 {
			limit = maxBytes
		}
	}

	limited := io.LimitReader(resp.Body, int64(limit)+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", errors.New("http read failed")
	}
	if len(body) > limit {
		return "", errors.New("http response too large")
	}
	return string(body), nil
}

type httpMaxBytesContextKey struct{}

func luaHeaders(raw lua.LValue) (map[string]string, error) {
	if raw == lua.LNil {
		return nil, nil
	}

	headersTable, ok := raw.(*lua.LTable)
	if !ok {
		return nil, errors.New("http headers must be an object")
	}

	headers := map[string]string{}
	var firstErr error
	headersTable.ForEach(func(key, value lua.LValue) {
		if firstErr != nil {
			return
		}

		name, ok := key.(lua.LString)
		if !ok {
			firstErr = errors.New("http header name must be a string")
			return
		}
		headerName := strings.TrimSpace(string(name))
		headerValue := strings.TrimSpace(value.String())
		if headerName == "" || headerValue == "" {
			firstErr = errors.New("http headers cannot be empty")
			return
		}
		headers[headerName] = headerValue
	})
	if firstErr != nil {
		return nil, firstErr
	}
	return headers, nil
}

func luaHeaderTable(l *lua.LState, headers http.Header) *lua.LTable {
	out := l.NewTable()
	for key, values := range headers {
		out.RawSetString(strings.ToLower(strings.TrimSpace(key)), lua.LString(strings.Join(values, ", ")))
	}
	return out
}

func luaTableString(table *lua.LTable, key string) string {
	if table == nil {
		return ""
	}
	value, ok := table.RawGetString(key).(lua.LString)
	if !ok {
		return ""
	}
	return string(value)
}

func luaTableMaxBytes(table *lua.LTable) (int, error) {
	if table == nil {
		return defaultPluginHTTPMaxBytes, nil
	}
	raw := table.RawGetString("max_bytes")
	if raw == lua.LNil {
		return defaultPluginHTTPMaxBytes, nil
	}
	number, ok := raw.(lua.LNumber)
	if !ok {
		return 0, errors.New("http max_bytes must be a number")
	}
	maxBytes := int(number)
	if float64(number) != float64(maxBytes) {
		return 0, errors.New("http max_bytes must be an integer")
	}
	if maxBytes < 1 {
		return 0, errors.New("http max_bytes must be positive")
	}
	if maxBytes > maxPluginHTTPMaxBytes {
		return 0, fmt.Errorf("http max_bytes must be <= %d", maxPluginHTTPMaxBytes)
	}
	return maxBytes, nil
}

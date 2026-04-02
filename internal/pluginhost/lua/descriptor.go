package luaplugin

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type RouteKind string

const (
	RouteCommand   RouteKind = "command"
	RouteComponent RouteKind = "component"
	RouteModal     RouteKind = "modal"
	RouteEvent     RouteKind = "event"
	RouteJob       RouteKind = "job"
)

type Definition struct {
	Commands   []CommandSpec
	Components []string
	Modals     []string
	Events     []string
	Jobs       []JobSpec
}

type CommandSpec struct {
	Name          string
	Description   string
	DescriptionID string
	Ephemeral     bool
	Options       []CommandOptionSpec
	Subcommands   []SubcommandSpec
	Groups        []CommandGroupSpec
}

type CommandOptionSpec struct {
	Name          string
	Type          string
	Description   string
	DescriptionID string
	Required      bool
	Choices       []OptionChoiceSpec
	MinValue      *float64
	MaxValue      *float64
	MinLength     *int
	MaxLength     *int
	ChannelTypes  []int
}

type SubcommandSpec struct {
	Name          string
	Description   string
	DescriptionID string
	Ephemeral     *bool
	Options       []CommandOptionSpec
}

type CommandGroupSpec struct {
	Name          string
	Description   string
	DescriptionID string
	Subcommands   []SubcommandSpec
}

type OptionChoiceSpec struct {
	Name  string
	Value any
}

type JobSpec struct {
	ID       string
	Schedule string
}

type pluginDefinition struct {
	meta       Definition
	commands   map[string]*lua.LFunction
	components map[string]*lua.LFunction
	modals     map[string]*lua.LFunction
	events     map[string]*lua.LFunction
	jobs       map[string]*lua.LFunction
}

func (d *pluginDefinition) lookup(kind RouteKind, routeID string) (*lua.LFunction, bool) {
	if d == nil {
		return nil, false
	}

	routeID = strings.TrimSpace(routeID)
	if routeID == "" {
		return nil, false
	}

	switch kind {
	case RouteCommand:
		fn, ok := d.commands[routeID]
		return fn, ok
	case RouteComponent:
		fn, ok := d.components[routeID]
		return fn, ok
	case RouteModal:
		fn, ok := d.modals[routeID]
		return fn, ok
	case RouteEvent:
		fn, ok := d.events[routeID]
		return fn, ok
	case RouteJob:
		fn, ok := d.jobs[routeID]
		return fn, ok
	default:
		return nil, false
	}
}

func (d *pluginDefinition) definition() Definition {
	if d == nil {
		return Definition{}
	}

	return Definition{
		Commands:   append([]CommandSpec(nil), d.meta.Commands...),
		Components: append([]string(nil), d.meta.Components...),
		Modals:     append([]string(nil), d.meta.Modals...),
		Events:     append([]string(nil), d.meta.Events...),
		Jobs:       append([]JobSpec(nil), d.meta.Jobs...),
	}
}

func parsePluginDefinition(raw lua.LValue) (*pluginDefinition, error) {
	if raw == lua.LNil {
		return nil, nil
	}

	root, ok := raw.(*lua.LTable)
	if !ok {
		return nil, errors.New("plugin entrypoint must return a table")
	}

	commands, commandHandlers, err := parseCommandEntries(root.RawGetString("commands"))
	if err != nil {
		return nil, fmt.Errorf("commands: %w", err)
	}
	components, componentHandlers, err := parseNamedRoutes(root.RawGetString("components"))
	if err != nil {
		return nil, fmt.Errorf("components: %w", err)
	}
	modals, modalHandlers, err := parseNamedRoutes(root.RawGetString("modals"))
	if err != nil {
		return nil, fmt.Errorf("modals: %w", err)
	}
	events, eventHandlers, err := parseNamedRoutes(root.RawGetString("events"))
	if err != nil {
		return nil, fmt.Errorf("events: %w", err)
	}
	jobs, jobHandlers, err := parseJobEntries(root.RawGetString("jobs"))
	if err != nil {
		return nil, fmt.Errorf("jobs: %w", err)
	}

	return &pluginDefinition{
		meta: Definition{
			Commands:   commands,
			Components: components,
			Modals:     modals,
			Events:     events,
			Jobs:       jobs,
		},
		commands:   commandHandlers,
		components: componentHandlers,
		modals:     modalHandlers,
		events:     eventHandlers,
		jobs:       jobHandlers,
	}, nil
}

func parseCommandEntries(raw lua.LValue) ([]CommandSpec, map[string]*lua.LFunction, error) {
	if raw == lua.LNil {
		return nil, nil, nil
	}

	list, err := requireArray(raw, "commands must be an array")
	if err != nil {
		return nil, nil, err
	}

	out := make([]CommandSpec, 0, len(list))
	handlers := make(map[string]*lua.LFunction, len(list))
	for idx, item := range list {
		table, ok := item.(*lua.LTable)
		if !ok {
			return nil, nil, fmt.Errorf("command %d must be an object", idx+1)
		}

		name := tableString(table, "name")
		if name == "" {
			return nil, nil, fmt.Errorf("command %d missing name", idx+1)
		}
		if _, exists := handlers[name]; exists {
			return nil, nil, fmt.Errorf("duplicate command %q", name)
		}

		handler, err := tableFunction(table, "run")
		if err != nil {
			return nil, nil, fmt.Errorf("command %q: %w", name, err)
		}

		spec, err := parseCommandTable(table)
		if err != nil {
			return nil, nil, fmt.Errorf("command %q: %w", name, err)
		}

		out = append(out, spec)
		handlers[name] = handler
	}

	return out, handlers, nil
}

func parseCommandTable(table *lua.LTable) (CommandSpec, error) {
	spec := CommandSpec{
		Name:          tableString(table, "name"),
		Description:   tableString(table, "description"),
		DescriptionID: tableString(table, "description_id"),
		Ephemeral:     tableBool(table, "ephemeral"),
	}
	if spec.Description == "" {
		return CommandSpec{}, errors.New("missing description")
	}

	options, err := parseOptionsField(table.RawGetString("options"))
	if err != nil {
		return CommandSpec{}, err
	}
	spec.Options = options

	subcommands, err := parseSubcommandsField(table.RawGetString("subcommands"))
	if err != nil {
		return CommandSpec{}, err
	}
	spec.Subcommands = subcommands

	groups, err := parseGroupsField(table.RawGetString("groups"))
	if err != nil {
		return CommandSpec{}, err
	}
	spec.Groups = groups

	return spec, nil
}

func parseSubcommandsField(raw lua.LValue) ([]SubcommandSpec, error) {
	if raw == lua.LNil {
		return nil, nil
	}

	list, err := requireArray(raw, "subcommands must be an array")
	if err != nil {
		return nil, err
	}

	out := make([]SubcommandSpec, 0, len(list))
	for idx, item := range list {
		table, ok := item.(*lua.LTable)
		if !ok {
			return nil, fmt.Errorf("subcommand %d must be an object", idx+1)
		}

		spec := SubcommandSpec{
			Name:          tableString(table, "name"),
			Description:   tableString(table, "description"),
			DescriptionID: tableString(table, "description_id"),
			Ephemeral:     tableOptionalBool(table, "ephemeral"),
		}
		if spec.Name == "" {
			return nil, fmt.Errorf("subcommand %d missing name", idx+1)
		}
		if spec.Description == "" {
			return nil, fmt.Errorf("subcommand %q missing description", spec.Name)
		}

		options, err := parseOptionsField(table.RawGetString("options"))
		if err != nil {
			return nil, fmt.Errorf("subcommand %q: %w", spec.Name, err)
		}
		spec.Options = options
		out = append(out, spec)
	}

	return out, nil
}

func parseGroupsField(raw lua.LValue) ([]CommandGroupSpec, error) {
	if raw == lua.LNil {
		return nil, nil
	}

	list, err := requireArray(raw, "groups must be an array")
	if err != nil {
		return nil, err
	}

	out := make([]CommandGroupSpec, 0, len(list))
	for idx, item := range list {
		table, ok := item.(*lua.LTable)
		if !ok {
			return nil, fmt.Errorf("group %d must be an object", idx+1)
		}

		spec := CommandGroupSpec{
			Name:          tableString(table, "name"),
			Description:   tableString(table, "description"),
			DescriptionID: tableString(table, "description_id"),
		}
		if spec.Name == "" {
			return nil, fmt.Errorf("group %d missing name", idx+1)
		}
		if spec.Description == "" {
			return nil, fmt.Errorf("group %q missing description", spec.Name)
		}

		subcommands, err := parseSubcommandsField(table.RawGetString("subcommands"))
		if err != nil {
			return nil, fmt.Errorf("group %q: %w", spec.Name, err)
		}
		spec.Subcommands = subcommands
		out = append(out, spec)
	}

	return out, nil
}

func parseOptionsField(raw lua.LValue) ([]CommandOptionSpec, error) {
	if raw == lua.LNil {
		return nil, nil
	}

	list, err := requireArray(raw, "options must be an array")
	if err != nil {
		return nil, err
	}

	out := make([]CommandOptionSpec, 0, len(list))
	for idx, item := range list {
		table, ok := item.(*lua.LTable)
		if !ok {
			return nil, fmt.Errorf("option %d must be an object", idx+1)
		}

		spec := CommandOptionSpec{
			Name:          tableString(table, "name"),
			Type:          tableString(table, "type"),
			Description:   tableString(table, "description"),
			DescriptionID: tableString(table, "description_id"),
			Required:      tableBool(table, "required"),
			MinValue:      tableOptionalFloat(table, "min_value"),
			MaxValue:      tableOptionalFloat(table, "max_value"),
			MinLength:     tableOptionalInt(table, "min_length"),
			MaxLength:     tableOptionalInt(table, "max_length"),
			ChannelTypes:  tableIntSlice(table, "channel_types"),
		}
		if spec.Name == "" {
			return nil, fmt.Errorf("option %d missing name", idx+1)
		}
		if spec.Type == "" {
			return nil, fmt.Errorf("option %q missing type", spec.Name)
		}
		if spec.Description == "" {
			return nil, fmt.Errorf("option %q missing description", spec.Name)
		}

		choices, err := parseChoicesField(table.RawGetString("choices"))
		if err != nil {
			return nil, fmt.Errorf("option %q: %w", spec.Name, err)
		}
		spec.Choices = choices
		out = append(out, spec)
	}

	return out, nil
}

func parseChoicesField(raw lua.LValue) ([]OptionChoiceSpec, error) {
	if raw == lua.LNil {
		return nil, nil
	}

	list, err := requireArray(raw, "choices must be an array")
	if err != nil {
		return nil, err
	}

	out := make([]OptionChoiceSpec, 0, len(list))
	for idx, item := range list {
		table, ok := item.(*lua.LTable)
		if !ok {
			return nil, fmt.Errorf("choice %d must be an object", idx+1)
		}

		name := tableString(table, "name")
		if name == "" {
			return nil, fmt.Errorf("choice %d missing name", idx+1)
		}
		value := table.RawGetString("value")
		if value == lua.LNil {
			return nil, fmt.Errorf("choice %q missing value", name)
		}
		scalar, err := luaChoiceValue(value)
		if err != nil {
			return nil, fmt.Errorf("choice %q: %w", name, err)
		}
		out = append(out, OptionChoiceSpec{Name: name, Value: scalar})
	}

	return out, nil
}

func parseNamedRoutes(raw lua.LValue) ([]string, map[string]*lua.LFunction, error) {
	if raw == lua.LNil {
		return nil, nil, nil
	}

	table, ok := raw.(*lua.LTable)
	if !ok {
		return nil, nil, errors.New("routes must be an object")
	}

	names := []string{}
	handlers := map[string]*lua.LFunction{}
	var firstErr error
	table.ForEach(func(key, value lua.LValue) {
		if firstErr != nil {
			return
		}

		name, ok := key.(lua.LString)
		if !ok {
			firstErr = errors.New("route key must be a string")
			return
		}
		routeID := strings.TrimSpace(string(name))
		if routeID == "" {
			firstErr = errors.New("route key cannot be empty")
			return
		}
		if _, exists := handlers[routeID]; exists {
			firstErr = fmt.Errorf("duplicate route %q", routeID)
			return
		}

		handler, err := routeFunction(value)
		if err != nil {
			firstErr = fmt.Errorf("route %q: %w", routeID, err)
			return
		}

		names = append(names, routeID)
		handlers[routeID] = handler
	})
	if firstErr != nil {
		return nil, nil, firstErr
	}

	sort.Strings(names)
	return names, handlers, nil
}

func parseJobEntries(raw lua.LValue) ([]JobSpec, map[string]*lua.LFunction, error) {
	if raw == lua.LNil {
		return nil, nil, nil
	}

	list, err := requireArray(raw, "jobs must be an array")
	if err != nil {
		return nil, nil, err
	}

	out := make([]JobSpec, 0, len(list))
	handlers := make(map[string]*lua.LFunction, len(list))
	for idx, item := range list {
		table, ok := item.(*lua.LTable)
		if !ok {
			return nil, nil, fmt.Errorf("job %d must be an object", idx+1)
		}

		id := tableString(table, "id")
		if id == "" {
			return nil, nil, fmt.Errorf("job %d missing id", idx+1)
		}
		if _, exists := handlers[id]; exists {
			return nil, nil, fmt.Errorf("duplicate job %q", id)
		}

		run, err := tableFunction(table, "run")
		if err != nil {
			return nil, nil, fmt.Errorf("job %q: %w", id, err)
		}
		schedule := tableString(table, "schedule")
		if schedule == "" {
			return nil, nil, fmt.Errorf("job %q missing schedule", id)
		}

		out = append(out, JobSpec{ID: id, Schedule: schedule})
		handlers[id] = run
	}

	return out, handlers, nil
}

func requireArray(raw lua.LValue, errMsg string) ([]lua.LValue, error) {
	table, ok := raw.(*lua.LTable)
	if !ok {
		return nil, errors.New(errMsg)
	}

	list := make([]lua.LValue, 0, table.Len())
	for idx := 1; idx <= table.Len(); idx++ {
		list = append(list, table.RawGetInt(idx))
	}
	return list, nil
}

func routeFunction(raw lua.LValue) (*lua.LFunction, error) {
	switch value := raw.(type) {
	case *lua.LFunction:
		return value, nil
	case *lua.LTable:
		return tableFunction(value, "run")
	default:
		return nil, errors.New("route must be a function or { run = function(...) end }")
	}
}

func tableFunction(table *lua.LTable, key string) (*lua.LFunction, error) {
	fn, ok := table.RawGetString(key).(*lua.LFunction)
	if !ok || fn == nil {
		return nil, fmt.Errorf("missing %s function", key)
	}
	return fn, nil
}

func tableString(table *lua.LTable, key string) string {
	value, ok := table.RawGetString(key).(lua.LString)
	if !ok {
		return ""
	}
	return strings.TrimSpace(string(value))
}

func tableBool(table *lua.LTable, key string) bool {
	value, ok := table.RawGetString(key).(lua.LBool)
	if !ok {
		return false
	}
	return bool(value)
}

func tableOptionalBool(table *lua.LTable, key string) *bool {
	value, ok := table.RawGetString(key).(lua.LBool)
	if !ok {
		return nil
	}
	out := bool(value)
	return &out
}

func tableOptionalFloat(table *lua.LTable, key string) *float64 {
	value, ok := table.RawGetString(key).(lua.LNumber)
	if !ok {
		return nil
	}
	out := float64(value)
	return &out
}

func tableOptionalInt(table *lua.LTable, key string) *int {
	value, ok := table.RawGetString(key).(lua.LNumber)
	if !ok {
		return nil
	}
	out := int(value)
	return &out
}

func tableIntSlice(table *lua.LTable, key string) []int {
	raw := table.RawGetString(key)
	list, err := requireArray(raw, "")
	if err != nil {
		return nil
	}

	out := make([]int, 0, len(list))
	for _, item := range list {
		number, ok := item.(lua.LNumber)
		if !ok {
			continue
		}
		out = append(out, int(number))
	}
	return out
}

func luaChoiceValue(raw lua.LValue) (any, error) {
	switch value := raw.(type) {
	case lua.LString:
		return string(value), nil
	case lua.LNumber:
		return float64(value), nil
	case lua.LBool:
		return bool(value), nil
	default:
		return nil, fmt.Errorf("unsupported choice value type %s", raw.Type().String())
	}
}

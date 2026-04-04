package pluginhost

import "github.com/xsyetopz/go-mamusiabtw/internal/pluginhost/lua"

func commandsFromDefinition(def luaplugin.Definition) []Command {
	out := make([]Command, 0, len(def.Commands))
	for _, command := range def.Commands {
		out = append(out, Command{
			Name:                     command.Name,
			Description:              command.Description,
			DescriptionID:            command.DescriptionID,
			Ephemeral:                command.Ephemeral,
			DefaultMemberPermissions: append([]string(nil), command.DefaultMemberPermissions...),
			Options:                  optionsFromDefinition(command.Options),
			Subcommands:              subcommandsFromDefinition(command.Subcommands),
			Groups:                   groupsFromDefinition(command.Groups),
		})
	}
	return out
}

func optionsFromDefinition(list []luaplugin.CommandOptionSpec) []CommandOption {
	out := make([]CommandOption, 0, len(list))
	for _, opt := range list {
		out = append(out, CommandOption{
			Name:          opt.Name,
			Type:          opt.Type,
			Description:   opt.Description,
			DescriptionID: opt.DescriptionID,
			Required:      opt.Required,
			Choices:       choicesFromDefinition(opt.Choices),
			MinValue:      opt.MinValue,
			MaxValue:      opt.MaxValue,
			MinLength:     opt.MinLength,
			MaxLength:     opt.MaxLength,
			ChannelTypes:  append([]int(nil), opt.ChannelTypes...),
		})
	}
	return out
}

func subcommandsFromDefinition(list []luaplugin.SubcommandSpec) []Subcommand {
	out := make([]Subcommand, 0, len(list))
	for _, subcommand := range list {
		out = append(out, Subcommand{
			Name:          subcommand.Name,
			Description:   subcommand.Description,
			DescriptionID: subcommand.DescriptionID,
			Ephemeral:     subcommand.Ephemeral,
			Options:       optionsFromDefinition(subcommand.Options),
		})
	}
	return out
}

func groupsFromDefinition(list []luaplugin.CommandGroupSpec) []CommandGroup {
	out := make([]CommandGroup, 0, len(list))
	for _, group := range list {
		out = append(out, CommandGroup{
			Name:          group.Name,
			Description:   group.Description,
			DescriptionID: group.DescriptionID,
			Subcommands:   subcommandsFromDefinition(group.Subcommands),
		})
	}
	return out
}

func choicesFromDefinition(list []luaplugin.OptionChoiceSpec) []OptionChoice {
	out := make([]OptionChoice, 0, len(list))
	for _, choice := range list {
		out = append(out, OptionChoice{
			Name:  choice.Name,
			Value: choice.Value,
		})
	}
	return out
}

func jobsFromDefinition(def luaplugin.Definition) []Job {
	out := make([]Job, 0, len(def.Jobs))
	for _, job := range def.Jobs {
		out = append(out, Job{
			ID:       job.ID,
			Schedule: job.Schedule,
		})
	}
	return out
}

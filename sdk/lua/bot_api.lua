---@meta

---@class MamusiaBtwGuildRef
---@field id string

---@class MamusiaBtwChannelRef
---@field id string

---@class MamusiaBtwUserRef
---@field id string

---@class MamusiaBtwAttachmentRef
---@field id string
---@field filename string
---@field url string
---@field size integer
---@field width? integer
---@field height? integer
---@field content_type? string

---@class MamusiaBtwPluginRef
---@field id string

---@class MamusiaBtwScopedStore
---@field get fun(key: string): (any|nil, boolean)
---@field put fun(key: string, value: any): boolean
---@field del fun(key: string): boolean
---@field get_json fun(key: string): (string|nil, boolean)
---@field put_json fun(key: string, value_json: string): boolean

---@class MamusiaBtwCommandContext
---@field name string
---@field kind 'slash'|'user'|'message'
---@field group string
---@field subcommand string
---@field args table<string, any>
---@field resolved table<string, table>

---@class MamusiaBtwAutocompleteContext
---@field command string
---@field group string
---@field subcommand string
---@field option string
---@field value any

---@class MamusiaBtwTargetContext
---@field user? MamusiaBtwUser
---@field member? MamusiaBtwMember
---@field message? MamusiaBtwMessageInfo

---@class MamusiaBtwComponentContext
---@field id string
---@field kind string
---@field values any[]|nil

---@class MamusiaBtwModalContext
---@field id string
---@field fields table<string, any>

---@class MamusiaBtwEventContext
---@field name string

---@class MamusiaBtwJobContext
---@field id string

---@class MamusiaBtwRouteContext
---@field guild_id string
---@field channel_id string
---@field user_id string
---@field locale string
---@field guild MamusiaBtwGuildRef
---@field channel MamusiaBtwChannelRef
---@field user MamusiaBtwUserRef
---@field plugin MamusiaBtwPluginRef
---@field store MamusiaBtwScopedStore
---@field options table<string, any>
---@field args table<string, any>|nil
---@field command MamusiaBtwCommandContext|nil
---@field component MamusiaBtwComponentContext|nil
---@field modal MamusiaBtwModalContext|nil
---@field event MamusiaBtwEventContext|nil
---@field job MamusiaBtwJobContext|nil
---@field target MamusiaBtwTargetContext|nil
---@field autocomplete MamusiaBtwAutocompleteContext|nil
---@field bot MamusiaBtwAPI

---@class MamusiaBtwPresent
---@field kind? 'info'|'success'|'warning'|'error'|'ok'|'warn'|'err'
---@field title? string
---@field body? string
---@field fields? { name: string, value: string, inline?: boolean }[]

---@class MamusiaBtwButton
---@field type 'button'
---@field id string
---@field label? string
---@field style? 'primary'|'secondary'|'success'|'danger'|'link'
---@field url? string
---@field disabled? boolean

---@class MamusiaBtwStringSelectOption
---@field label string
---@field value string
---@field description? string

---@class MamusiaBtwStringSelect
---@field type 'string_select'
---@field id string
---@field placeholder? string
---@field min_values? integer
---@field max_values? integer
---@field disabled? boolean
---@field options MamusiaBtwStringSelectOption[]

---@class MamusiaBtwUserSelect
---@field type 'user_select'
---@field id string
---@field placeholder? string
---@field min_values? integer
---@field max_values? integer
---@field disabled? boolean

---@class MamusiaBtwRoleSelect
---@field type 'role_select'
---@field id string
---@field placeholder? string
---@field min_values? integer
---@field max_values? integer
---@field disabled? boolean

---@class MamusiaBtwMentionableSelect
---@field type 'mentionable_select'
---@field id string
---@field placeholder? string
---@field min_values? integer
---@field max_values? integer
---@field disabled? boolean

---@class MamusiaBtwChannelSelect
---@field type 'channel_select'
---@field id string
---@field placeholder? string
---@field min_values? integer
---@field max_values? integer
---@field disabled? boolean
---@field channel_types? integer[]

---@alias MamusiaBtwComponent MamusiaBtwButton|MamusiaBtwStringSelect|MamusiaBtwUserSelect|MamusiaBtwRoleSelect|MamusiaBtwMentionableSelect|MamusiaBtwChannelSelect

---@class MamusiaBtwEmbedField
---@field name string
---@field value string
---@field inline? boolean

---@class MamusiaBtwEmbed
---@field title? string
---@field description? string
---@field url? string
---@field color? integer
---@field image_url? string
---@field thumbnail_url? string
---@field footer? string|{ text: string, icon_url?: string }
---@field author? { name: string, url?: string, icon_url?: string }
---@field fields? MamusiaBtwEmbedField[]

---@class MamusiaBtwModalField
---@field id string
---@field label string
---@field description? string
---@field style? 'short'|'paragraph'
---@field required? boolean
---@field placeholder? string
---@field value? string
---@field min_length? integer
---@field max_length? integer

---@class MamusiaBtwResponseBase
---@field ephemeral? boolean
---@field content? string
---@field embeds? MamusiaBtwEmbed[]
---@field components? MamusiaBtwComponent[][]

---@class MamusiaBtwMessageResponse: MamusiaBtwResponseBase
---@field type? 'message'|'update'
---@field present? MamusiaBtwPresent

---@class MamusiaBtwModalResponse
---@field type 'modal'
---@field id string
---@field title string
---@field components MamusiaBtwModalField[]

---@alias MamusiaBtwResponse MamusiaBtwMessageResponse|MamusiaBtwModalResponse

---@class MamusiaBtwCommandChoice
---@field name string
---@field value string|number|boolean

---@class MamusiaBtwCommandOption
---@field name string
---@field type 'string'|'bool'|'int'|'float'|'user'|'channel'|'role'|'mentionable'|'attachment'
---@field description string
---@field description_id? string
---@field required? boolean
---@field autocomplete? string
---@field choices? MamusiaBtwCommandChoice[]
---@field min_value? number
---@field max_value? number
---@field min_length? integer
---@field max_length? integer
---@field channel_types? integer[]

---@class MamusiaBtwSubcommand
---@field name string
---@field description string
---@field description_id? string
---@field ephemeral? boolean
---@field options? MamusiaBtwCommandOption[]

---@class MamusiaBtwCommandGroup
---@field name string
---@field description string
---@field description_id? string
---@field subcommands MamusiaBtwSubcommand[]

---@class MamusiaBtwCommandRoute
---@field type? 'slash'
---@field name string
---@field description string
---@field description_id? string
---@field ephemeral? boolean
---@field default_member_permissions? string[]
---@field options? MamusiaBtwCommandOption[]
---@field subcommands? MamusiaBtwSubcommand[]
---@field groups? MamusiaBtwCommandGroup[]
---@field run fun(ctx: MamusiaBtwRouteContext): MamusiaBtwResponse|nil

---@class MamusiaBtwUserCommandRoute
---@field type 'user'
---@field name string
---@field default_member_permissions? string[]
---@field run fun(ctx: MamusiaBtwRouteContext): MamusiaBtwResponse|nil

---@class MamusiaBtwMessageCommandRoute
---@field type 'message'
---@field name string
---@field default_member_permissions? string[]
---@field run fun(ctx: MamusiaBtwRouteContext): MamusiaBtwResponse|nil

---@class MamusiaBtwJobRoute
---@field id string
---@field schedule string
---@field run fun(ctx: MamusiaBtwRouteContext): table|nil

---@class MamusiaBtwPluginDefinition
---@field commands? MamusiaBtwCommandRoute[]
---@field user_commands? MamusiaBtwUserCommandRoute[]
---@field message_commands? MamusiaBtwMessageCommandRoute[]
---@field autocompletes? table<string, fun(ctx: MamusiaBtwRouteContext): MamusiaBtwCommandChoice[]|{ choices: MamusiaBtwCommandChoice[] }|nil>
---@field components? table<string, fun(ctx: MamusiaBtwRouteContext): MamusiaBtwResponse|nil>
---@field modals? table<string, fun(ctx: MamusiaBtwRouteContext): MamusiaBtwResponse|nil>
---@field events? table<string, fun(ctx: MamusiaBtwRouteContext): table|nil>
---@field jobs? MamusiaBtwJobRoute[]

---@class MamusiaBtwLogAPI
---@field info fun(msg: string)

---@class MamusiaBtwI18nAPI
---@field t fun(message_id: string, data: table|nil, plural_count: any|nil): string

---@class MamusiaBtwStoreAPI
---@field get fun(guild_id: string, key: string): (any|nil, boolean)
---@field put fun(guild_id: string, key: string, value: any): boolean
---@field del fun(guild_id: string, key: string): boolean
---@field get_json fun(guild_id: string, key: string): (string|nil, boolean)
---@field put_json fun(guild_id: string, key: string, value_json: string): boolean

---@class MamusiaBtwOptionAPI
---@field string fun(name: string, spec: table): MamusiaBtwCommandOption
---@field bool fun(name: string, spec: table): MamusiaBtwCommandOption
---@field int fun(name: string, spec: table): MamusiaBtwCommandOption
---@field float fun(name: string, spec: table): MamusiaBtwCommandOption
---@field user fun(name: string, spec: table): MamusiaBtwCommandOption
---@field channel fun(name: string, spec: table): MamusiaBtwCommandOption
---@field role fun(name: string, spec: table): MamusiaBtwCommandOption
---@field mentionable fun(name: string, spec: table): MamusiaBtwCommandOption
---@field attachment fun(name: string, spec: table): MamusiaBtwCommandOption

---@class MamusiaBtwUIAPI
---@field reply fun(spec: table): MamusiaBtwMessageResponse
---@field defer fun(spec?: { ephemeral?: boolean }): (boolean, string|nil)
---@field update fun(spec: table): MamusiaBtwMessageResponse
---@field modal fun(id: string, spec: table): MamusiaBtwModalResponse
---@field present fun(spec: table): MamusiaBtwMessageResponse
---@field button fun(id: string, spec: table): MamusiaBtwButton
---@field choice fun(name: string, value: string|number|boolean): MamusiaBtwCommandChoice
---@field choices fun(list: MamusiaBtwCommandChoice[]): MamusiaBtwCommandChoice[]
---@field string_option fun(label: string, value: string, spec?: table): MamusiaBtwStringSelectOption
---@field string_select fun(id: string, spec: table): MamusiaBtwStringSelect
---@field text_input fun(id: string, spec: table): MamusiaBtwModalField

---@class MamusiaBtwEffectsAPI
---Automation-only effects for event/job handlers.
---@field send_channel fun(spec: { channel_id?: string, message: MamusiaBtwResponse|string }): table
---@field send_dm fun(spec: { user_id?: string, message: MamusiaBtwResponse|string }): table
---@field timeout_member fun(spec: { guild_id?: string, user_id?: string, until_unix: integer }): table

---@class MamusiaBtwDiscordSendResult
---@field message_id string
---@field channel_id string
---@field user_id? string

---@class MamusiaBtwRole
---@field id string|integer
---@field name string
---@field mention string
---@field color integer
---@field hoist boolean
---@field mentionable boolean
---@field position integer
---@field managed boolean
---@field permissions integer
---@field created_at integer

---@class MamusiaBtwUser
---@field id string|integer
---@field username string
---@field display_name string
---@field mention string
---@field bot boolean
---@field system boolean
---@field accent_color integer
---@field avatar_url string
---@field banner_url string
---@field created_at integer

---@class MamusiaBtwMember
---@field user_id string|integer
---@field guild_id string|integer
---@field joined_at integer
---@field role_ids (string|integer)[]
---@field avatar_url string
---@field banner_url string

---@class MamusiaBtwGuild
---@field id string|integer
---@field name string
---@field description string
---@field owner_id string|integer
---@field roles_count integer
---@field emojis_count integer
---@field stickers_count integer
---@field member_count integer
---@field channels_count integer
---@field icon_url string
---@field banner_url string
---@field created_at integer

---@class MamusiaBtwChannel
---@field id string|integer
---@field name string
---@field mention string
---@field type string
---@field parent_id? string|integer
---@field permissions integer
---@field created_at integer

---@class MamusiaBtwMessageInfo
---@field id string|integer
---@field channel_id string|integer
---@field author_id string|integer
---@field content string
---@field created_at integer
---@field edited_at? integer
---@field pinned? boolean

---@class MamusiaBtwDiscordMessagesAPI
---@field get fun(spec: { channel_id?: string|integer, message_id: string|integer }): (MamusiaBtwMessageInfo|nil, string|nil)
---@field list fun(spec: { channel_id?: string|integer, around_message_id?: string|integer, before_message_id?: string|integer, after_message_id?: string|integer, limit: integer }): (MamusiaBtwMessageInfo[]|nil, string|nil)
---@field delete fun(spec: { channel_id?: string|integer, message_id: string|integer }): (boolean, string|nil)
---@field bulk_delete fun(spec: { channel_id?: string|integer, message_ids: (string|integer)[] }): ({ deleted_count: integer }|nil, string|nil)
---@field purge fun(spec: { channel_id?: string|integer, mode: "all"|"before"|"after"|"around", anchor_message_id?: string|integer, count: integer }): ({ deleted_count: integer }|nil, string|nil)
---@field crosspost fun(spec: { channel_id?: string|integer, message_id: string|integer }): (MamusiaBtwMessageInfo|nil, string|nil)
---@field pin fun(spec: { channel_id?: string|integer, message_id: string|integer }): (boolean, string|nil)
---@field unpin fun(spec: { channel_id?: string|integer, message_id: string|integer }): (boolean, string|nil)

---@class MamusiaBtwDiscordReactionsAPI
---@field list fun(spec: { channel_id?: string|integer, message_id: string|integer, emoji: string, after_user_id?: string|integer, limit?: integer }): (MamusiaBtwUser[]|nil, string|nil)
---@field add fun(spec: { channel_id?: string|integer, message_id: string|integer, emoji: string }): (boolean, string|nil)
---@field remove_own fun(spec: { channel_id?: string|integer, message_id: string|integer, emoji: string }): (boolean, string|nil)
---@field remove_user fun(spec: { channel_id?: string|integer, message_id: string|integer, user_id?: string|integer, emoji: string }): (boolean, string|nil)
---@field clear fun(spec: { channel_id?: string|integer, message_id: string|integer }): (boolean, string|nil)
---@field clear_for_emoji fun(spec: { channel_id?: string|integer, message_id: string|integer, emoji: string }): (boolean, string|nil)

---@class MamusiaBtwEmoji
---@field id string|integer
---@field name string

---@class MamusiaBtwSticker
---@field id string|integer
---@field name string

---@class MamusiaBtwDiscordUsersAPI
---@field self fun(): (MamusiaBtwUser|nil, string|nil)
---@field get fun(spec?: { user_id?: string|integer }): (MamusiaBtwUser|nil, string|nil)

---@class MamusiaBtwDiscordGuildsAPI
---@field get fun(spec?: { guild_id?: string|integer }): (MamusiaBtwGuild|nil, string|nil)
---@field list_invites fun(spec: { guild_id?: string|integer }): (MamusiaBtwInvite[]|nil, string|nil)

---@class MamusiaBtwDiscordChannelsAPI
---@field get fun(spec?: { channel_id?: string|integer }): (MamusiaBtwChannel|nil, string|nil)
---@field create fun(spec: { guild_id?: string|integer, name: string, type?: string, topic?: string, parent_id?: string|integer, nsfw?: boolean, slowmode?: integer, position?: integer, bitrate?: integer, user_limit?: integer }): (MamusiaBtwChannel|nil, string|nil)
---@field edit fun(spec: { channel_id?: string|integer, name?: string, topic?: string, parent_id?: string|integer, nsfw?: boolean, slowmode?: integer, position?: integer, bitrate?: integer, user_limit?: integer }): (MamusiaBtwChannel|nil, string|nil)
---@field delete fun(spec: { channel_id?: string|integer }): (boolean, string|nil)
---@field set_slowmode fun(spec: { channel_id?: string|integer, seconds: integer }): (boolean, string|nil)
---@field set_overwrite fun(spec: { channel_id?: string|integer, target_id: string|integer, target_type: 'role'|'member'|'user', allow?: integer, deny?: integer }): (boolean, string|nil)
---@field delete_overwrite fun(spec: { channel_id?: string|integer, target_id: string|integer }): (boolean, string|nil)
---@field list_invites fun(spec: { channel_id?: string|integer }): (MamusiaBtwInvite[]|nil, string|nil)
---@field list_webhooks fun(spec: { channel_id?: string|integer }): (MamusiaBtwWebhook[]|nil, string|nil)

---@class MamusiaBtwDiscordMembersAPI
---@field get fun(spec?: { guild_id?: string|integer, user_id?: string|integer }): (MamusiaBtwMember|nil, string|nil)
---@field timeout fun(spec: { guild_id?: string, user_id?: string, until_unix: integer }): (boolean, string|nil)
---@field set_nickname fun(spec: { guild_id?: string|integer, user_id?: string|integer, nickname?: string }): (boolean, string|nil)

---@class MamusiaBtwDiscordRolesAPI
---@field get fun(spec: { guild_id?: string|integer, role_id: string|integer }): (MamusiaBtwRole|nil, string|nil)
---@field create fun(spec: { guild_id?: string|integer, name: string, color?: integer, hoist?: boolean, mentionable?: boolean }): (MamusiaBtwRole|nil, string|nil)
---@field edit fun(spec: { guild_id?: string|integer, role_id: string|integer, name?: string, color?: integer, hoist?: boolean, mentionable?: boolean }): (MamusiaBtwRole|nil, string|nil)
---@field delete fun(spec: { guild_id?: string|integer, role_id: string|integer }): (boolean, string|nil)
---@field add_to_member fun(spec: { guild_id?: string|integer, user_id?: string|integer, role_id: string|integer }): (boolean, string|nil)
---@field remove_from_member fun(spec: { guild_id?: string|integer, user_id?: string|integer, role_id: string|integer }): (boolean, string|nil)

---@class MamusiaBtwThread
---@field id string|integer
---@field guild_id string|integer
---@field parent_id string|integer
---@field name string
---@field mention string
---@field type string
---@field archived boolean
---@field locked boolean
---@field auto_archive_duration integer
---@field created_at integer

---@class MamusiaBtwDiscordThreadsAPI
---@field create_from_message fun(spec: { channel_id?: string|integer, message_id: string|integer, name: string, auto_archive_duration?: integer, slowmode?: integer }): (MamusiaBtwThread|nil, string|nil)
---@field create_in_channel fun(spec: { channel_id?: string|integer, name: string, type?: string, auto_archive_duration?: integer, invitable?: boolean }): (MamusiaBtwThread|nil, string|nil)
---@field join fun(spec: { thread_id?: string|integer }): (boolean, string|nil)
---@field leave fun(spec: { thread_id?: string|integer }): (boolean, string|nil)
---@field add_member fun(spec: { thread_id?: string|integer, user_id?: string|integer }): (boolean, string|nil)
---@field remove_member fun(spec: { thread_id?: string|integer, user_id?: string|integer }): (boolean, string|nil)
---@field update fun(spec: { thread_id?: string|integer, name?: string, archived?: boolean, locked?: boolean, invitable?: boolean, auto_archive_duration?: integer, slowmode?: integer }): (MamusiaBtwThread|nil, string|nil)

---@class MamusiaBtwInvite
---@field code string
---@field url string
---@field guild_id string|integer
---@field channel_id string|integer
---@field inviter_id string|integer
---@field max_age integer
---@field max_uses integer
---@field uses integer
---@field temporary boolean
---@field created_at integer

---@class MamusiaBtwDiscordInvitesAPI
---@field create fun(spec: { channel_id?: string|integer, max_age?: integer, max_uses?: integer, temporary?: boolean, unique?: boolean }): (MamusiaBtwInvite|nil, string|nil)
---@field get fun(spec: { code: string }): (MamusiaBtwInvite|nil, string|nil)
---@field delete fun(spec: { code: string }): (boolean, string|nil)
---@field list_channel fun(spec: { channel_id?: string|integer }): (MamusiaBtwInvite[]|nil, string|nil)
---@field list_guild fun(spec: { guild_id?: string|integer }): (MamusiaBtwInvite[]|nil, string|nil)

---@class MamusiaBtwWebhook
---@field id string|integer
---@field guild_id string|integer
---@field channel_id string|integer
---@field application_id string|integer
---@field name string
---@field token string
---@field url string

---@class MamusiaBtwDiscordWebhooksAPI
---@field create fun(spec: { channel_id?: string|integer, name: string }): (MamusiaBtwWebhook|nil, string|nil)
---@field get fun(spec: { webhook_id: string|integer }): (MamusiaBtwWebhook|nil, string|nil)
---@field list_channel fun(spec: { channel_id?: string|integer }): (MamusiaBtwWebhook[]|nil, string|nil)
---@field edit fun(spec: { webhook_id: string|integer, name?: string, channel_id?: string|integer }): (MamusiaBtwWebhook|nil, string|nil)
---@field delete fun(spec: { webhook_id: string|integer }): (boolean, string|nil)
---@field execute fun(spec: { webhook_id: string|integer, token: string }, message: MamusiaBtwResponse|string): (MamusiaBtwDiscordSendResult|nil, string|nil)

---@class MamusiaBtwDiscordEmojisAPI
---@field create fun(spec: { guild_id?: string|integer, name: string, file: MamusiaBtwAttachmentRef }): (MamusiaBtwEmoji|nil, string|nil)
---@field edit fun(spec: { guild_id?: string|integer, emoji: string, name: string }): (MamusiaBtwEmoji|nil, string|nil)
---@field delete fun(spec: { guild_id?: string|integer, emoji: string }): (boolean, string|nil)

---@class MamusiaBtwDiscordStickersAPI
---@field create fun(spec: { guild_id?: string|integer, name: string, description?: string, emoji_tag: string, file: MamusiaBtwAttachmentRef }): (MamusiaBtwSticker|nil, string|nil)
---@field edit fun(spec: { guild_id?: string|integer, id: string, name: string, description?: string }): (MamusiaBtwSticker|nil, string|nil)
---@field delete fun(spec: { guild_id?: string|integer, id: string }): (boolean, string|nil)

---@class MamusiaBtwDiscordAPI
---@field self_user fun(): (MamusiaBtwUser|nil, string|nil)
---@field get_user fun(spec?: { user_id?: string|integer }): (MamusiaBtwUser|nil, string|nil)
---@field get_member fun(spec?: { guild_id?: string|integer, user_id?: string|integer }): (MamusiaBtwMember|nil, string|nil)
---@field get_guild fun(spec?: { guild_id?: string|integer }): (MamusiaBtwGuild|nil, string|nil)
---@field get_role fun(spec: { guild_id?: string|integer, role_id: string|integer }): (MamusiaBtwRole|nil, string|nil)
---@field get_channel fun(spec?: { channel_id?: string|integer }): (MamusiaBtwChannel|nil, string|nil)
---@field get_message fun(spec: { channel_id?: string|integer, message_id: string|integer }): (MamusiaBtwMessageInfo|nil, string|nil)
---@field send_dm fun(spec: { user_id?: string, message: MamusiaBtwResponse|string }): (MamusiaBtwDiscordSendResult|nil, string|nil)
---@field send_channel fun(spec: { channel_id?: string, message: MamusiaBtwResponse|string }): (MamusiaBtwDiscordSendResult|nil, string|nil)
---@field timeout_member fun(spec: { guild_id?: string, user_id?: string, until_unix: integer }): (boolean, string|nil)
---@field set_slowmode fun(spec: { channel_id?: string|integer, seconds: integer }): (boolean, string|nil)
---@field set_nickname fun(spec: { guild_id?: string|integer, user_id?: string|integer, nickname?: string }): (boolean, string|nil)
---@field create_role fun(spec: { guild_id?: string|integer, name: string, color?: integer, hoist?: boolean, mentionable?: boolean }): (MamusiaBtwRole|nil, string|nil)
---@field edit_role fun(spec: { guild_id?: string|integer, role_id: string|integer, name?: string, color?: integer, hoist?: boolean, mentionable?: boolean }): (MamusiaBtwRole|nil, string|nil)
---@field delete_role fun(spec: { guild_id?: string|integer, role_id: string|integer }): (boolean, string|nil)
---@field add_role fun(spec: { guild_id?: string|integer, user_id?: string|integer, role_id: string|integer }): (boolean, string|nil)
---@field remove_role fun(spec: { guild_id?: string|integer, user_id?: string|integer, role_id: string|integer }): (boolean, string|nil)
---@field list_messages fun(spec: { channel_id?: string|integer, around_message_id?: string|integer, before_message_id?: string|integer, after_message_id?: string|integer, limit: integer }): (MamusiaBtwMessageInfo[]|nil, string|nil)
---@field delete_message fun(spec: { channel_id?: string|integer, message_id: string|integer }): (boolean, string|nil)
---@field bulk_delete_messages fun(spec: { channel_id?: string|integer, message_ids: (string|integer)[] }): ({ deleted_count: integer }|nil, string|nil)
---@field purge_messages fun(spec: { channel_id?: string|integer, mode: "all"|"before"|"after"|"around", anchor_message_id?: string|integer, count: integer }): ({ deleted_count: integer }|nil, string|nil)
---@field crosspost_message fun(spec: { channel_id?: string|integer, message_id: string|integer }): (MamusiaBtwMessageInfo|nil, string|nil)
---@field pin_message fun(spec: { channel_id?: string|integer, message_id: string|integer }): (boolean, string|nil)
---@field unpin_message fun(spec: { channel_id?: string|integer, message_id: string|integer }): (boolean, string|nil)
---@field get_reactions fun(spec: { channel_id?: string|integer, message_id: string|integer, emoji: string, after_user_id?: string|integer, limit?: integer }): (MamusiaBtwUser[]|nil, string|nil)
---@field add_reaction fun(spec: { channel_id?: string|integer, message_id: string|integer, emoji: string }): (boolean, string|nil)
---@field remove_own_reaction fun(spec: { channel_id?: string|integer, message_id: string|integer, emoji: string }): (boolean, string|nil)
---@field remove_user_reaction fun(spec: { channel_id?: string|integer, message_id: string|integer, user_id?: string|integer, emoji: string }): (boolean, string|nil)
---@field clear_reactions fun(spec: { channel_id?: string|integer, message_id: string|integer }): (boolean, string|nil)
---@field clear_reactions_for_emoji fun(spec: { channel_id?: string|integer, message_id: string|integer, emoji: string }): (boolean, string|nil)
---@field messages MamusiaBtwDiscordMessagesAPI
---@field reactions MamusiaBtwDiscordReactionsAPI
---@field users MamusiaBtwDiscordUsersAPI
---@field guilds MamusiaBtwDiscordGuildsAPI
---@field channels MamusiaBtwDiscordChannelsAPI
---@field members MamusiaBtwDiscordMembersAPI
---@field roles MamusiaBtwDiscordRolesAPI
---@field threads MamusiaBtwDiscordThreadsAPI
---@field invites MamusiaBtwDiscordInvitesAPI
---@field webhooks MamusiaBtwDiscordWebhooksAPI
---@field emojis MamusiaBtwDiscordEmojisAPI
---@field stickers MamusiaBtwDiscordStickersAPI
---@field create_emoji fun(spec: { guild_id?: string|integer, name: string, file: MamusiaBtwAttachmentRef }): (MamusiaBtwEmoji|nil, string|nil)
---@field edit_emoji fun(spec: { guild_id?: string|integer, emoji: string, name: string }): (MamusiaBtwEmoji|nil, string|nil)
---@field delete_emoji fun(spec: { guild_id?: string|integer, emoji: string }): (boolean, string|nil)
---@field create_sticker fun(spec: { guild_id?: string|integer, name: string, description?: string, emoji_tag: string, file: MamusiaBtwAttachmentRef }): (MamusiaBtwSticker|nil, string|nil)
---@field edit_sticker fun(spec: { guild_id?: string|integer, id: string, name: string, description?: string }): (MamusiaBtwSticker|nil, string|nil)
---@field delete_sticker fun(spec: { guild_id?: string|integer, id: string }): (boolean, string|nil)

---@class MamusiaBtwRandomAPI
---@field int fun(min: integer, max: integer): integer
---@field choice fun(list: any[]): any

---@class MamusiaBtwTimeAPI
---@field unix fun(): integer

---@class MamusiaBtwRuntimeAPI
---@field build_info fun(): { version: string, description: string, repository: string, mascot_image_url: string, developer_url: string, support_server_url: string }

---@class MamusiaBtwHTTPResponse
---@field status integer
---@field body string
---@field headers table<string, string>

---@class MamusiaBtwHTTPAPI
---@field get fun(spec: { url: string, headers?: table<string, string>, max_bytes?: integer }): MamusiaBtwHTTPResponse
---@field get_json fun(spec: { url: string, headers?: table<string, string>, max_bytes?: integer }): any

---@class MamusiaBtwUserSettings
---@field user_id integer
---@field timezone string
---@field dm_channel_id string
---@field created_at integer
---@field updated_at integer

---@class MamusiaBtwUserSettingsAPI
---@field normalize_timezone fun(timezone: string): string|nil
---@field get fun(user_id?: string|integer): (MamusiaBtwUserSettings|nil, boolean)
---@field set_timezone fun(user_id: string|integer, timezone: string): string
---@field clear_timezone fun(user_id?: string|integer): boolean

---@class MamusiaBtwCheckIn
---@field id string
---@field user_id integer
---@field mood integer
---@field created_at integer

---@class MamusiaBtwCheckInsAPI
---@field create fun(spec: { user_id?: string|integer, mood: integer, created_at?: integer }): MamusiaBtwCheckIn
---@field list fun(user_id?: string|integer, limit?: integer): MamusiaBtwCheckIn[]

---@class MamusiaBtwReminder
---@field id string
---@field user_id integer
---@field schedule string
---@field kind string
---@field note string
---@field delivery string
---@field guild_id string
---@field channel_id string
---@field enabled boolean
---@field next_run_at integer
---@field last_run_at integer|nil
---@field failure_count integer
---@field created_at integer
---@field updated_at integer

---@class MamusiaBtwReminderPlan
---@field schedule string
---@field next_run_at integer

---@class MamusiaBtwRemindersAPI
---@field plan fun(spec: { user_id?: string|integer, schedule: string }): MamusiaBtwReminderPlan|nil
---@field create fun(spec: { user_id?: string|integer, schedule: string, kind: string, note?: string, delivery?: string, guild_id?: string|integer, channel_id?: string|integer }): MamusiaBtwReminder|nil
---@field list fun(user_id?: string|integer, limit?: integer): MamusiaBtwReminder[]
---@field delete fun(user_id: string|integer, reminder_id: string): boolean

---@class MamusiaBtwWarning
---@field id string
---@field guild_id integer
---@field user_id integer
---@field moderator_id integer
---@field reason string
---@field created_at integer

---@class MamusiaBtwWarningsAPI
---@field count fun(guild_id?: string|integer, user_id?: string|integer): integer
---@field list fun(guild_id?: string|integer, user_id?: string|integer, limit?: integer): MamusiaBtwWarning[]
---@field create fun(spec: { id?: string, guild_id?: string|integer, user_id?: string|integer, moderator_id?: string|integer, reason: string, created_at?: integer }): MamusiaBtwWarning
---@field delete fun(warning_id: string): boolean

---@class MamusiaBtwAuditAPI
---@field append fun(spec: { guild_id?: string|integer, actor_id?: string|integer, action: string, target_type?: 'user'|'guild', target_id?: string|integer, created_at?: integer, meta_json?: string }): boolean

---@class MamusiaBtwAPI
---@field log MamusiaBtwLogAPI
---@field i18n MamusiaBtwI18nAPI
---@field store MamusiaBtwStoreAPI
---@field usersettings MamusiaBtwUserSettingsAPI
---@field checkins MamusiaBtwCheckInsAPI
---@field reminders MamusiaBtwRemindersAPI
---@field warnings MamusiaBtwWarningsAPI
---@field audit MamusiaBtwAuditAPI
---@field option MamusiaBtwOptionAPI
---@field ui MamusiaBtwUIAPI
---@field effects MamusiaBtwEffectsAPI
---@field discord MamusiaBtwDiscordAPI
---@field runtime MamusiaBtwRuntimeAPI
---@field random MamusiaBtwRandomAPI
---@field time MamusiaBtwTimeAPI
---@field http MamusiaBtwHTTPAPI
---@field plugin fun(spec: MamusiaBtwPluginDefinition): MamusiaBtwPluginDefinition
---@field command fun(name: string, spec: table): MamusiaBtwCommandRoute
---@field user_command fun(name: string, spec: table): MamusiaBtwUserCommandRoute
---@field message_command fun(name: string, spec: table): MamusiaBtwMessageCommandRoute
---@field job fun(id: string, spec: table): MamusiaBtwJobRoute
---@field require fun(path: string): any
---@field include fun(path: string): boolean

---@type MamusiaBtwAPI
bot = bot

---Legacy host API alias kept for older plugins.
---@type table
mamusiabtw = mamusiabtw

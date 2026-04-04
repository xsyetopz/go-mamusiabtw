---@meta

---@class MamusiaBtwGuildRef
---@field id string

---@class MamusiaBtwChannelRef
---@field id string

---@class MamusiaBtwUserRef
---@field id string

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
---@field group string
---@field subcommand string
---@field args table<string, any>
---@field resolved table<string, table>

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
---@field footer? string
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
---@field name string
---@field description string
---@field description_id? string
---@field ephemeral? boolean
---@field default_member_permissions? string[]
---@field options? MamusiaBtwCommandOption[]
---@field subcommands? MamusiaBtwSubcommand[]
---@field groups? MamusiaBtwCommandGroup[]
---@field run fun(ctx: MamusiaBtwRouteContext): MamusiaBtwResponse|nil

---@class MamusiaBtwJobRoute
---@field id string
---@field schedule string
---@field run fun(ctx: MamusiaBtwRouteContext): table|nil

---@class MamusiaBtwPluginDefinition
---@field commands? MamusiaBtwCommandRoute[]
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
---@field update fun(spec: table): MamusiaBtwMessageResponse
---@field modal fun(id: string, spec: table): MamusiaBtwModalResponse
---@field present fun(spec: table): MamusiaBtwMessageResponse
---@field button fun(id: string, spec: table): MamusiaBtwButton
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

---@class MamusiaBtwDiscordAPI
---@field send_dm fun(spec: { user_id?: string, message: MamusiaBtwResponse|string }): (MamusiaBtwDiscordSendResult|nil, string|nil)
---@field send_channel fun(spec: { channel_id?: string, message: MamusiaBtwResponse|string }): (MamusiaBtwDiscordSendResult|nil, string|nil)
---@field timeout_member fun(spec: { guild_id?: string, user_id?: string, until_unix: integer }): (boolean, string|nil)

---@class MamusiaBtwRandomAPI
---@field int fun(min: integer, max: integer): integer
---@field choice fun(list: any[]): any

---@class MamusiaBtwTimeAPI
---@field unix fun(): integer

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
---@field random MamusiaBtwRandomAPI
---@field time MamusiaBtwTimeAPI
---@field http MamusiaBtwHTTPAPI
---@field plugin fun(spec: MamusiaBtwPluginDefinition): MamusiaBtwPluginDefinition
---@field command fun(name: string, spec: table): MamusiaBtwCommandRoute
---@field job fun(id: string, spec: table): MamusiaBtwJobRoute
---@field require fun(path: string): any
---@field include fun(path: string): boolean

---@type MamusiaBtwAPI
bot = bot

---Legacy host API alias kept for older plugins.
---@type table
mamusiabtw = mamusiabtw

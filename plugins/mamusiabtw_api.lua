---@meta

---@class MamusiaBtwAPI
---@field log fun(msg: string)
---@field include fun(path: string): boolean
---@field t fun(message_id: string, data: table|nil, plural_count: any|nil): string
---@field kv_get fun(guild_id: string, key: string): (any|nil, boolean)
---@field kv_put fun(guild_id: string, key: string, value: any): boolean
---@field kv_del fun(guild_id: string, key: string): boolean
---@field kv_get_json fun(guild_id: string, key: string): (string|nil, boolean)
---@field kv_put_json fun(guild_id: string, key: string, value_json: string): boolean

---Global injected by the host at runtime.
---@type MamusiaBtwAPI
mamusiabtw = mamusiabtw

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
---@field present? MamusiaBtwPresent

---@class MamusiaBtwMessageResponse: MamusiaBtwResponseBase
---@field type? 'message'|'update'
---@field components? MamusiaBtwComponent[][]

---@class MamusiaBtwModalResponse
---@field type 'modal'
---@field id? string
---@field title? string
---@field components MamusiaBtwModalField[]

---@alias MamusiaBtwResponse MamusiaBtwMessageResponse|MamusiaBtwModalResponse

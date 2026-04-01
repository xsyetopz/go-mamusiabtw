-- Example plugin.
-- Shows scoped KV usage and safe logging via the host API.

local function get_counter(guild_id)
  local v, ok = jagpda.kv_get(guild_id, "counter")
  if ok and type(v) == "number" then
    return v
  end
  return 0
end

local function set_counter(guild_id, n)
  jagpda.kv_put(guild_id, "counter", n)
end

local function render(n, msg_type)
  return {
    type = msg_type or "message",
    ephemeral = true,
    content = jagpda.t("example.counter", { Count = n }, nil),
    components = {
      {
        { type = "button", id = "inc", label = "Increment", style = "primary" },
        { type = "button", id = "set", label = "Set…", style = "secondary" }
      }
    }
  }
end

function Handle(cmd, ctx)
  jagpda.log("Handle " .. cmd)

  local guild_id = ctx.guild_id
  if guild_id == "" then
    return {
      present = {
        kind = "error",
        title = jagpda.t("example.not_in_guild.title", nil, nil),
        body = jagpda.t("example.not_in_guild.body", nil, nil)
      },
      ephemeral = true
    }
  end

  local n = get_counter(guild_id) + 1
  set_counter(guild_id, n)
  return render(n, "message")
end

function HandleComponent(id, ctx)
  local guild_id = ctx.guild_id
  if guild_id == "" then
    return { type = "update", content = "This must be used in a server." }
  end

  if id == "inc" then
    local n = get_counter(guild_id) + 1
    set_counter(guild_id, n)
    return render(n, "update")
  end

  if id == "set" then
    return {
      type = "modal",
      id = "set_counter",
      title = jagpda.t("example.set.title", nil, nil),
      components = {
        { id = "value", label = jagpda.t("example.set.label", nil, nil), style = "short", required = true, placeholder = "123" }
      }
    }
  end

  return nil
end

function HandleModal(id, ctx)
  if id ~= "set_counter" then
    return nil
  end

  local guild_id = ctx.guild_id
  if guild_id == "" then
    return { type = "message", ephemeral = true, content = "This must be used in a server." }
  end

  local fields = ctx.options.fields or {}
  local raw = fields.value
  local n = tonumber(raw)
  if n == nil then
    return {
      present = {
        kind = "error",
        title = jagpda.t("example.invalid.title", nil, nil),
        body = jagpda.t("example.invalid.body", { Raw = tostring(raw) }, nil)
      },
      ephemeral = true
    }
  end

  set_counter(guild_id, n)
  return render(n, "update")
end

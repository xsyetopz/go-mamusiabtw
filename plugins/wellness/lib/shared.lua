local ui = bot.ui

local M = {}

M.reminder_kinds = {
  "hydrate",
  "stretch",
  "breathe",
  "meds",
  "sleep",
  "checkin",
}

M.reminder_kind_choices = {
  { name = "hydrate", value = "hydrate" },
  { name = "stretch", value = "stretch" },
  { name = "breathe", value = "breathe" },
  { name = "meds",    value = "meds" },
  { name = "sleep",   value = "sleep" },
  { name = "checkin", value = "checkin" },
}

M.delivery_choices = {
  { name = "dm",      value = "dm" },
  { name = "channel", value = "channel" },
}

function M.reply_text(message, ephemeral)
  return ui.reply({
    ephemeral = ephemeral,
    content = message,
  })
end

function M.update_text(message)
  return ui.update({
    content = message,
    components = {},
  })
end

function M.trim(value)
  if value == nil then
    return ""
  end
  return (tostring(value):gsub("^%s+", ""):gsub("%s+$", ""))
end

function M.timestamp(unix_seconds)
  return "<t:" .. tostring(unix_seconds) .. ":f>"
end

function M.ensure_guild(ctx, message)
  if ctx.guild.id ~= "" then
    return nil
  end
  return M.reply_text(message, true)
end

function M.reminder_option_label(reminder)
  local kind = M.trim(reminder.kind)
  if kind == "" then
    kind = reminder.id
  end
  local label = kind .. " • " .. M.timestamp(reminder.next_run_at)
  if #label > 90 then
    label = label:sub(1, 90)
  end
  return label
end

return M

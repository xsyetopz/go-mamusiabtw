local ui = bot.ui
local option = bot.option
local random = bot.random
local kawaii = bot.kawaii
local i18n = bot.i18n

local FUN_COLOR = 0x5865F2
local FUN_WARN_COLOR = 0xFEE75C
local FUN_ERROR_COLOR = 0xED4245

local function mention(user_id)
  return "<@" .. tostring(user_id) .. ">"
end

local function embed(spec)
  return {
    description = spec.description,
    color = spec.color,
    image_url = spec.image_url,
    footer = spec.footer,
  }
end

local function reply_embed(spec)
  return ui.reply({
    ephemeral = spec.ephemeral,
    content = spec.content,
    embeds = { embed(spec) }
  })
end

local function ensure_guild(ctx)
  if ctx.guild.id ~= "" then
    return nil
  end
  return ui.reply({
    ephemeral = true,
    content = "This command must be used in a server."
  })
end

local function ensure_other_user(ctx, target_id)
  if tostring(target_id) ~= tostring(ctx.user.id) then
    return nil
  end
  return ui.reply({
    ephemeral = true,
    content = i18n.t("fun.kawaii.self_error", nil, nil)
  })
end

local function is_allowed_dice_sides(sides)
  return sides == 4 or sides == 6 or sides == 8 or sides == 10 or sides == 12 or sides == 20
end

local function roll_notation(number, sides, modifier)
  local base = tostring(number) .. "d" .. tostring(sides)
  if modifier > 0 then
    return base .. "+" .. tostring(modifier)
  end
  if modifier < 0 then
    return base .. tostring(modifier)
  end
  return base
end

local function is_open_ended_question(question)
  if question == nil then
    return false
  end
  if #question < 3 then
    return false
  end
  local last = question:sub(-1)
  return last == "?" or last == "." or last == "!"
end

local function eight_ball_answers()
  return {
    "It is certain.",
    "It is decidedly so.",
    "Without a doubt.",
    "Yes - definitely.",
    "You may rely on it.",
    "As I see it, yes.",
    "Most likely.",
    "Outlook good.",
    "Yes.",
    "Signs point to yes.",
    "Reply hazy, try again.",
    "Ask again later.",
    "Better not tell you now.",
    "Cannot predict now.",
    "Concentrate and ask again.",
    "Don't count on it.",
    "My reply is no.",
    "My sources say no.",
    "Outlook not so good.",
    "Very doubtful."
  }
end

local function endpoint_emoji(endpoint)
  if endpoint == "hug" then
    return "🤗"
  end
  if endpoint == "pat" then
    return "🫳"
  end
  if endpoint == "poke" then
    return "👉"
  end
  if endpoint == "shrug" then
    return "🤷"
  end
  return "✨"
end

local function fetch_kawaii_or_error(endpoint)
  local ok, result = pcall(kawaii.gif, endpoint)
  if ok then
    return result, nil
  end
  return nil, reply_embed({
    color = FUN_ERROR_COLOR,
    description = i18n.t("fun.kawaii.error", nil, nil),
  })
end

local function kawaii_command(name, description, description_id, option_name_id, option_desc_id, endpoint)
  return bot.command(name, {
    description = description,
    description_id = description_id,
    options = {
      option.user("user", {
        description = "Target user",
        description_id = option_desc_id,
        required = true
      })
    },
    run = function(ctx)
      local guild_error = ensure_guild(ctx)
      if guild_error ~= nil then
        return guild_error
      end

      local target_id = tostring(ctx.command.args.user or "")
      local self_error = ensure_other_user(ctx, target_id)
      if self_error ~= nil then
        return self_error
      end

      local gif_url, kawaii_error = fetch_kawaii_or_error(endpoint)
      if kawaii_error ~= nil then
        return kawaii_error
      end
      return reply_embed({
        color = FUN_COLOR,
        description = i18n.t("fun.kawaii.user_mention_only", {
          Emoji = endpoint_emoji(endpoint),
          User = mention(target_id),
        }, nil),
        image_url = gif_url,
        footer = i18n.t("fun.kawaii.footer", nil, nil),
      })
    end
  })
end

return bot.plugin({
  commands = {
    bot.command("flip", {
      description = "Flip a coin.",
      description_id = "cmd.flip.desc",
      run = function(ctx)
        local result = "heads"
        if random.int(0, 1) == 0 then
          result = "tails"
        end
        return reply_embed({
          color = FUN_COLOR,
          description = i18n.t("fun.flip.result", {
            User = mention(ctx.user.id),
            Result = result,
          }, nil),
        })
      end
    }),

    bot.command("roll", {
      description = "Roll some dice.",
      description_id = "cmd.roll.desc",
      options = {
        option.int("number", {
          description = "How many dice?",
          description_id = "cmd.roll.opt.number.desc",
          required = true,
          min_value = 1,
          max_value = 99,
        }),
        option.int("sides", {
          description = "How many sides per die?",
          description_id = "cmd.roll.opt.sides.desc",
          required = true,
          min_value = 4,
          max_value = 20,
        }),
        option.int("modifier", {
          description = "Any modifier to add?",
          description_id = "cmd.roll.opt.modifier.desc",
          min_value = -99,
          max_value = 99,
        }),
      },
      run = function(ctx)
        local number = tonumber(ctx.command.args.number or 0) or 0
        local sides = tonumber(ctx.command.args.sides or 0) or 0
        local modifier = tonumber(ctx.command.args.modifier or 0) or 0

        if not is_allowed_dice_sides(sides) then
          return reply_embed({
            ephemeral = true,
            color = FUN_WARN_COLOR,
            description = i18n.t("fun.roll.invalid_sides", { Sides = sides }, nil),
          })
        end

        local rolls = {}
        local sum = 0
        for _ = 1, number do
          local value = random.int(1, sides)
          table.insert(rolls, value)
          sum = sum + value
        end

        local verbose = table.concat(rolls, ", ")
        if modifier > 0 then
          verbose = verbose .. " + " .. tostring(modifier)
        elseif modifier < 0 then
          verbose = verbose .. " - " .. tostring(-modifier)
        end

        return reply_embed({
          color = FUN_COLOR,
          description = i18n.t("fun.roll.result", {
            User = mention(ctx.user.id),
            Notation = roll_notation(number, sides, modifier),
            Total = sum + modifier,
          }, nil),
          footer = verbose,
        })
      end
    }),

    bot.command("8ball", {
      description = "Ask the Magic 8 Ball a question.",
      description_id = "cmd.8ball.desc",
      options = {
        option.string("question", {
          description = "What do you want to ask?",
          description_id = "cmd.8ball.opt.question.desc",
          required = true,
          min_length = 3,
          max_length = 255,
        })
      },
      run = function(ctx)
        local question = tostring(ctx.command.args.question or "")
        if not is_open_ended_question(question) then
          return reply_embed({
            ephemeral = true,
            color = FUN_ERROR_COLOR,
            description = i18n.t("fun.8ball.question_error", { Question = question }, nil),
          })
        end

        return reply_embed({
          color = FUN_COLOR,
          description = i18n.t("fun.8ball.result", {
            Answer = random.choice(eight_ball_answers()),
            User = mention(ctx.user.id),
          }, nil),
        })
      end
    }),

    kawaii_command("hug", "Give someone a warm hug.", "cmd.hug.desc", "cmd.hug.opt.user.name", "cmd.hug.opt.user.desc",
      "hug"),
    kawaii_command("pat", "Give someone a gentle head-pat.", "cmd.pat.desc", "cmd.pat.opt.user.name",
      "cmd.pat.opt.user.desc", "pat"),
    kawaii_command("poke", "Give someone a tiny poke.", "cmd.poke.desc", "cmd.poke.opt.user.name",
      "cmd.poke.opt.user.desc", "poke"),

    bot.command("shrug", {
      description = "Send a shrug.",
      description_id = "cmd.shrug.desc",
      options = {
        option.string("message", {
          description = "Anything to add?",
          description_id = "cmd.shrug.opt.message.desc",
          max_length = 2000,
        })
      },
      run = function(ctx)
        local guild_error = ensure_guild(ctx)
        if guild_error ~= nil then
          return guild_error
        end

        local content = tostring(ctx.command.args.message or "")
        if content == "" then
          content = nil
        end

        local gif_url, kawaii_error = fetch_kawaii_or_error("shrug")
        if kawaii_error ~= nil then
          return kawaii_error
        end

        return ui.reply({
          content = content,
          embeds = {
            {
              color = FUN_COLOR,
              image_url = gif_url,
              footer = i18n.t("fun.kawaii.footer", nil, nil),
            }
          }
        })
      end
    }),
  }
})

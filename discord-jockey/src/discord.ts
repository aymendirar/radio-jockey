import { Client, Events, IntentsBitField, REST, Routes } from "discord.js";
import { logger } from "./util/logger.js";
import { registerPingCommand } from "./discord/command/ping.js";
import { registerPlayCommand } from "./discord/command/play.js";
import { registerSkipCommand } from "./discord/command/skip.js";
import { registerQueueCommand } from "./discord/command/queue.js";
import { registerRemoveCommand } from "./discord/command/remove.js";

export async function startDiscordBot(apiKey: string) {
  const bot = new Client({
    intents: [
      IntentsBitField.Flags.Guilds,
      IntentsBitField.Flags.GuildVoiceStates,
    ],
  });
  setupEventHandlers(bot);
  await bot.login(apiKey);
  return bot;
}

function setupEventHandlers(bot: Client) {
  handleLogin(bot);
  handleSlashCommands(bot);
}

function handleLogin(bot: Client) {
  bot.on(Events.ClientReady, (readyClient) => {
    logger.info("discord jockey spinning...", { ready: bot.isReady(), userTag: readyClient.user.tag });
  });
}

async function handleSlashCommands(bot: Client) {
  bot.on(Events.InteractionCreate, async (interaction) => {
    if (!interaction.isChatInputCommand()) return;

    console.log("received interaction:", interaction.commandName);
    try {
      await registerPingCommand(interaction);
      await registerPlayCommand(interaction);
      await registerSkipCommand(interaction);
      await registerQueueCommand(interaction);
      await registerRemoveCommand(interaction);
    } catch (err) {
      logger.error("unhandled error in interaction handler", { command: interaction.commandName, err });
    }
  });
}

export async function registerCommands(apiKey: string, botId: string) {
  const rest = new REST({ version: "10" }).setToken(apiKey);
  const commands = [
    {
      name: "ping",
      description: "Replies with Pong!",
    },
    {
      name: "play",
      description: "Add a track to the queue and start playing.",
      options: [
        {
          name: "url",
          description: "YouTube URL of the track to add",
          type: 3, // STRING
          required: false,
        },
      ],
    },
    {
      name: "skip",
      description: "Skip the current track.",
    },
    {
      name: "queue",
      description: "Show the current queue.",
    },
    {
      name: "remove",
      description: "Remove a track from the queue by position.",
      options: [
        {
          name: "position",
          description: "Position in the queue (1-based)",
          type: 4, // INTEGER
          required: true,
        },
      ],
    },
  ];

  try {
    logger.info("Started refreshing application (/) commands.");
    await rest.put(Routes.applicationCommands(botId), { body: commands });
    logger.info("Successfully reloaded application (/) commands.");
  } catch (error) {
    logger.error(`${error}`);
  }
}

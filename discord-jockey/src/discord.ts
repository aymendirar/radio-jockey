import { ApplicationCommandOptionType, Client, Events, IntentsBitField, REST, Routes } from "discord.js";
import { logger } from "./util/logger.js";
import { handlePingCommand } from "./discord/command/ping.js";
import { handlePlayCommand } from "./discord/command/play.js";
import { handleSkipCommand } from "./discord/command/skip.js";
import { handleQueueCommand } from "./discord/command/queue.js";
import { handleRemoveCommand } from "./discord/command/remove.js";
import { handleStopCommand } from "./discord/command/stop.js";

export async function startDiscordBot(apiKey: string) {
  const bot = new Client({
    intents: [
      IntentsBitField.Flags.Guilds,
      IntentsBitField.Flags.GuildVoiceStates,
    ],
  });

  bot.on(Events.ClientReady, (readyClient) => {
    logger.info("discord jockey spinning...", { ready: bot.isReady(), userTag: readyClient.user.tag });
  });

  bot.on(Events.InteractionCreate, async (interaction) => {
    if (!interaction.isChatInputCommand()) return;

    logger.info("received interaction", { command: interaction.commandName });
    try {
      switch (interaction.commandName) {
        case "ping": await handlePingCommand(interaction); break;
        case "play": await handlePlayCommand(interaction); break;
        case "skip": await handleSkipCommand(interaction); break;
        case "queue": await handleQueueCommand(interaction); break;
        case "remove": await handleRemoveCommand(interaction); break;
        case "stop": await handleStopCommand(interaction); break;
      }
    } catch (err) {
      logger.error("unhandled error in interaction handler", { command: interaction.commandName, err });
    }
  });

  await bot.login(apiKey);
  return bot;
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
          type: ApplicationCommandOptionType.String,
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
          type: ApplicationCommandOptionType.Integer,
          required: true,
        },
      ],
    },
    {
      name: "stop",
      description: "Stop playback and leave the voice channel.",
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

import { Client, Events, GatewayIntentBits, REST, Routes } from "discord.js";
import { logger } from "./util/logger";
import { registerPingCommand } from "./discord/command/ping";
import { registerPlayCommand } from "./discord/command/play";

export async function startDiscordBot(apiKey: string) {
  const bot = new Client({ intents: [GatewayIntentBits.Guilds, GatewayIntentBits.GuildVoiceStates] });
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
    logger
      .withMetadata({ ready: bot.isReady() })
      .withMetadata({ userTag: readyClient.user.tag })
      .info("discord jockey spinning...");
  });
}

async function handleSlashCommands(bot: Client) {
  bot.on(Events.InteractionCreate, async (interaction) => {
    if (!interaction.isChatInputCommand()) return;

    await registerPingCommand(interaction);
    await registerPlayCommand(interaction);
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
      description: "Join voice channel and play queue!",
    },
  ];

  try {
    logger.info("Started refreshing application (/) commands.");

    await rest.put(Routes.applicationCommands(botId), {
      body: commands,
    });

    logger.info("Successfully reloaded application (/) commands.");
  } catch (error) {
    logger.error(`${error}`);
  }
}

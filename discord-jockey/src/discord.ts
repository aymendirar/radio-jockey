import { Client, Events, GatewayIntentBits, REST, Routes } from "discord.js";
import { logger } from "./util/logger";
import { registerPingCommand } from "./discord/command/ping";

export async function startDiscordBot(apiKey: string) {
  const client = new Client({ intents: [GatewayIntentBits.Guilds] });
  setupEventHandlers(client);
  await client.login(apiKey);

  return client;
}

function setupEventHandlers(client: Client) {
  handleLogin(client);
  handleSlashCommands(client);
}

function handleLogin(client: Client) {
  client.on(Events.ClientReady, (readyClient) => {
    logger
      .withMetadata({ ready: client.isReady() })
      .withMetadata({ userTag: readyClient.user.tag })
      .info("discord jockey spinning...");
  });
}

async function handleSlashCommands(client: Client) {
  client.on(Events.InteractionCreate, async (interaction) => {
    if (!interaction.isChatInputCommand()) return;

    await registerPingCommand(interaction);
  });
}

export async function registerCommands(apiKey: string, clientId: string) {
  const rest = new REST({ version: "10" }).setToken(apiKey);
  const commands = [
    {
      name: "ping",
      description: "Replies with Pong!",
    },
  ];

  try {
    logger.info("Started refreshing application (/) commands.");

    await rest.put(Routes.applicationCommands(clientId), {
      body: commands,
    });

    logger.info("Successfully reloaded application (/) commands.");
  } catch (error) {
    logger.error(`${error}`);
  }
}

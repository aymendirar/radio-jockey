import { ChatInputCommandInteraction, type CacheType } from "discord.js";
import { radioClient } from "../../connect/client.js";
import { logger } from "../../util/logger.js";

export async function registerPingCommand(
  interaction: ChatInputCommandInteraction<CacheType>,
) {
  if (interaction.commandName !== "ping") return;

  logger.info("ping command received");
  const response = await radioClient.ping({});
  logger.info("ping response received", response.message);
  await interaction.reply(response.message);
}

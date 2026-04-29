import { ChatInputCommandInteraction, type CacheType } from "discord.js";
import { radioClient } from "../../connect/client";

export async function registerPingCommand(
  interaction: ChatInputCommandInteraction<CacheType>,
) {
  if (interaction.commandName === "ping") {
    const response = await radioClient.ping({});
    await interaction.reply(response.message);
  }
}

import { ChatInputCommandInteraction, type CacheType } from "discord.js";
import { client } from "../../connect/client";

export async function registerPingCommand(
  interaction: ChatInputCommandInteraction<CacheType>,
) {
  if (interaction.commandName === "ping") {
    const response = await client.ping({});
    await interaction.reply(response.message);
  }
}

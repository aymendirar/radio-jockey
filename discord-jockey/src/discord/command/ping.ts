import { ChatInputCommandInteraction, type CacheType } from "discord.js";

export async function registerPingCommand(
  interaction: ChatInputCommandInteraction<CacheType>,
) {
  if (interaction.commandName === "ping") {
    await interaction.reply("Pong!");
  }
}

import { ChatInputCommandInteraction, type CacheType } from "discord.js";
import { Code } from "@connectrpc/connect";
import { radioClient } from "../../connect/client.js";
import { withConnectError } from "../../util/helpers.js";
import { logger } from "../../util/logger.js";

export async function registerRemoveCommand(
  interaction: ChatInputCommandInteraction<CacheType>,
) {
  if (interaction.commandName !== "remove") return;

  const sessionId = interaction.guildId!;
  const index = interaction.options.getInteger("position", true) - 1;
  logger.withMetadata({ sessionId, position: index + 1 }).info("remove command received");

  await withConnectError(
    async () => {
      await radioClient.removeTrack({ sessionId, index });
      logger.withMetadata({ sessionId, position: index + 1 }).info("track removed");
      await interaction.reply(`Removed track at position ${index + 1}.`);
    },
    async (err) => {
      if (err.code === Code.InvalidArgument) {
        logger.withMetadata({ sessionId, position: index + 1 }).info("remove failed: invalid position");
        await interaction.reply("Invalid position.");
      } else {
        logger.withMetadata({ sessionId, err }).error("remove failed");
        await interaction.reply("Something went wrong.");
      }
    },
  );
}

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
  logger.info("remove command received", { sessionId, position: index + 1 });

  await withConnectError(
    async () => {
      await radioClient.removeTrack({ sessionId, index });
      logger.info("track removed", { sessionId, position: index + 1 });
      await interaction.reply(`Removed track at position ${index + 1}.`);
    },
    async (err) => {
      switch (err.code) {
        case Code.NotFound:
          logger.info("remove failed: session not found", { sessionId });
          await interaction.reply("No active session. Use /play to start one!");
          break;
        case Code.InvalidArgument:
          logger.info("remove failed: invalid position", { sessionId, position: index + 1 });
          await interaction.reply(index === 0 ? "Can't remove the currently playing track. Use /skip instead." : "Invalid position.");
          break;
        default:
          logger.error("remove failed", { sessionId, err });
          await interaction.reply("Something went wrong.");
      }
    },
  );
}

import { ChatInputCommandInteraction, type CacheType } from "discord.js";
import { Code } from "@connectrpc/connect";
import { radioClient } from "../../connect/client.js";
import { withConnectError } from "../../util/helpers.js";
import { logger } from "../../util/logger.js";

export async function registerSkipCommand(
  interaction: ChatInputCommandInteraction<CacheType>,
) {
  if (interaction.commandName !== "skip") return;

  const sessionId = interaction.guildId!;
  logger.withMetadata({ sessionId }).info("skip command received");

  await withConnectError(
    async () => {
      await radioClient.skipTrack({ sessionId });
      logger.withMetadata({ sessionId }).info("track skipped");
      await interaction.reply("Skipped!");
    },
    async (err) => {
      switch (err.code) {
        case Code.NotFound:
          logger.withMetadata({ sessionId }).info("skip failed: session not found");
          await interaction.reply("No active session. Use /play to start one!");
          break;
        case Code.FailedPrecondition:
          logger.withMetadata({ sessionId }).info("skip failed: queue empty");
          await interaction.reply("The queue is empty.");
          break;
        default:
          logger.withMetadata({ sessionId, err }).error("skip failed");
          await interaction.reply("Something went wrong.");
      }
    },
  );
}

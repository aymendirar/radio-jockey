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
  logger.info("skip command received", { sessionId });

  await withConnectError(
    async () => {
      await radioClient.skipTrack({ sessionId });
      logger.info("track skipped", { sessionId });
      await interaction.reply("Skipped!");
    },
    async (err) => {
      switch (err.code) {
        case Code.NotFound:
          logger.info("skip failed: session not found", { sessionId });
          await interaction.reply("No active session. Use /play to start one!");
          break;
        case Code.FailedPrecondition:
          logger.info("skip failed: queue empty", { sessionId });
          await interaction.reply("The queue is empty.");
          break;
        default:
          logger.error("skip failed", { sessionId, err });
          await interaction.reply("Something went wrong.");
      }
    },
  );
}

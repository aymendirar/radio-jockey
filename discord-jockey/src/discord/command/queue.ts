import { ChatInputCommandInteraction, type CacheType } from "discord.js";
import { Code } from "@connectrpc/connect";
import { radioClient } from "../../connect/client.js";
import { withConnectError } from "../../util/helpers.js";
import { logger } from "../../util/logger.js";

export async function registerQueueCommand(
  interaction: ChatInputCommandInteraction<CacheType>,
) {
  if (interaction.commandName !== "queue") return;

  const sessionId = interaction.guildId!;
  logger.withMetadata({ sessionId }).info("queue command received");

  await withConnectError(
    async () => {
      const res = await radioClient.listQueue({ sessionId });
      logger.withMetadata({ sessionId, count: res.tracks.length }).info("queue fetched");
      if (res.tracks.length === 0) {
        await interaction.reply("The queue is empty.");
        return;
      }
      const list = res.tracks
        .map((t, i) => `${i + 1}. **${t.title}** by **${t.artist}**`)
        .join("\n");
      await interaction.reply(`**Queue:**\n${list}`);
    },
    async (err) => {
      switch (err.code) {
        case Code.NotFound:
          logger.withMetadata({ sessionId }).info("queue fetch failed: session not found");
          await interaction.reply("No active session. Use /play to start one!");
          break;
        default:
          logger.withMetadata({ sessionId, err }).error("queue fetch failed");
          await interaction.reply("Something went wrong fetching the queue.");
      }
    },
  );
}

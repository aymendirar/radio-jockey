import { ChatInputCommandInteraction, type CacheType } from "discord.js";
import { radioClient } from "../../connect/client.js";
import { withConnectError } from "../../util/helpers.js";
import { logger } from "../../util/logger.js";

export async function registerNowPlayingCommand(
  interaction: ChatInputCommandInteraction<CacheType>,
) {
  if (interaction.commandName !== "now_playing") return;

  const sessionId = interaction.guildId!;
  logger.withMetadata({ sessionId }).info("now_playing command received");

  await withConnectError(
    async () => {
      const res = await radioClient.listQueue({ sessionId });
      const first = res.tracks[0];
      if (!first) {
        logger.withMetadata({ sessionId }).info("now_playing: nothing playing");
        await interaction.reply("Nothing is playing right now.");
        return;
      }
      logger.withMetadata({ sessionId, title: first.title, artist: first.artist }).info("now_playing: track found");
      await interaction.reply(`Now playing: **${first.title}** by **${first.artist}**`);
    },
    async (err) => {
      logger.withMetadata({ sessionId, err }).error("now_playing failed");
      await interaction.reply("Something went wrong.");
    },
  );
}

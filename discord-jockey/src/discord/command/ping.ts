import { ChatInputCommandInteraction, type CacheType } from "discord.js";
import { radioClient } from "../../connect/client.js";
import { withConnectError } from "../../util/helpers.js";
import { logger } from "../../util/logger.js";

export async function handlePingCommand(
  interaction: ChatInputCommandInteraction<CacheType>,
) {
  logger.info("ping command received");
  await withConnectError(
    async () => {
      const response = await radioClient.ping({});
      logger.info("ping response received");
      await interaction.reply(response.message);
    },
    async (err) => {
      logger.error("ping failed", { err });
      await interaction.reply("Something went wrong.");
    },
  );
}

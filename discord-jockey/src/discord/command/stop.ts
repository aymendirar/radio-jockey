import { ChatInputCommandInteraction, type CacheType } from "discord.js";
import { Code } from "@connectrpc/connect";
import { deleteSession } from "../../connect/auth/auth.js";
import { getSessionId, withConnectError } from "../../util/helpers.js";
import { logger } from "../../util/logger.js";
import { destroySession } from "../sessions.js";

export async function handleStopCommand(
  interaction: ChatInputCommandInteraction<CacheType>,
) {
  const sessionId = getSessionId(interaction);
  logger.info("stop command received", { sessionId });

  await withConnectError(
    async () => {
      await deleteSession(sessionId);
      destroySession(sessionId);
      logger.info("session stopped", { sessionId });
      await interaction.reply("Stopped and left the voice channel.");
    },
    async (err) => {
      switch (err.code) {
        case Code.NotFound:
          destroySession(sessionId);
          logger.info("stop: session not found, cleaning up locally", { sessionId });
          await interaction.reply("No active session.");
          break;
        default:
          logger.error("stop failed", { sessionId, err });
          await interaction.reply("Something went wrong.");
      }
    },
  );
}

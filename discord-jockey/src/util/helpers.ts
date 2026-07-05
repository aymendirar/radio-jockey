import { ConnectError } from "@connectrpc/connect";
import type { ChatInputCommandInteraction, CacheType } from "discord.js";

export function getSessionId(interaction: ChatInputCommandInteraction<CacheType>): string {
  const serverName = interaction.guild?.name ?? "unknown";
  const slug = serverName.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "");
  return `${slug}-${interaction.guildId}`;
}

export async function withConnectError<T, E = T>(
  fn: () => Promise<T>,
  onError: (err: ConnectError) => E | Promise<E>
): Promise<T | E> {
  try {
    return await fn();
  } catch (err) {
    if (err instanceof ConnectError) {
      return onError(err);
    }
    throw err;
  }
}

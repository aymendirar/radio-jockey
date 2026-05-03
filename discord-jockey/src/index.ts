import "dotenv/config";
import { startDiscordBot, registerCommands } from "./discord.js";

const { DISCORD_API_KEY, DISCORD_CLIENT_ID } = process.env;

await registerCommands(DISCORD_API_KEY!, DISCORD_CLIENT_ID!);
await startDiscordBot(DISCORD_API_KEY!);

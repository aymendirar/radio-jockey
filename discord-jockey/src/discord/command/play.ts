import {
  createAudioPlayer,
  AudioPlayerStatus,
  createAudioResource,
  entersState,
  StreamType,
  joinVoiceChannel,
  VoiceConnectionStatus,
} from "@discordjs/voice";
import {
  ChatInputCommandInteraction,
  GuildMember,
  type CacheType,
  type VoiceBasedChannel,
} from "discord.js";
import { Code } from "@connectrpc/connect";
import { radioClient } from "../../connect/client.js";
import { withConnectError } from "../../util/helpers.js";
import { logger } from "../../util/logger.js";
import { PassThrough } from "node:stream";
import http from "node:http";

const KILOBYTE = 1024;
const MEGABYTE = KILOBYTE * KILOBYTE;
const BUFFER_SIZE = 20;

async function connectToChannel(channel: VoiceBasedChannel) {
  const connection = joinVoiceChannel({
    channelId: channel.id,
    guildId: channel.guild.id,
    adapterCreator: channel.guild.voiceAdapterCreator,
  });
  try {
    await entersState(connection, VoiceConnectionStatus.Ready, 30_000);
    return connection;
  } catch (error) {
    connection.destroy();
    throw error;
  }
}

async function getOrCreateSession(sessionId: string): Promise<string> {
  return withConnectError(
    async () => {
      const res = await radioClient.createSession({ sessionId });
      logger.info("session created", { sessionId, streamUrl: res.streamUrl });
      return res.streamUrl;
    },
    async (err) => {
      switch (err.code) {
        case Code.AlreadyExists: {
          const res = await radioClient.getSession({ sessionId });
          logger.info("session already exists", { sessionId, streamUrl: res.streamUrl });
          return res.streamUrl;
        }
        default:
          throw err;
      }
    },
  );
}

export async function registerPlayCommand(
  interaction: ChatInputCommandInteraction<CacheType>,
) {
  if (interaction.commandName !== "play") return;
  const player = createAudioPlayer();

  if (!interaction.inGuild()) {
    await interaction.reply("This command can only be used in a server!");
    return;
  }

  const member = interaction.member as GuildMember;
  if (!member.voice.channel) {
    await interaction.reply("Join a voice channel then try again!");
    return;
  }
  const voiceChannel = member.voice.channel;

  const trackUrl = interaction.options.getString("url");
  const sessionId = interaction.guildId!;
  logger.info("play command received", { sessionId, trackUrl });

  let streamUrl: string;
  try {
    streamUrl = await getOrCreateSession(sessionId);
  } catch (err) {
    logger.error("failed to get or create session", { sessionId, err });
    await interaction.reply("Failed to start a session. Please try again.");
    return;
  }

  if (trackUrl) {
    await interaction.reply(`Adding **${trackUrl}** to the queue...`);
    await withConnectError(
      async () => {
        const { track } = await radioClient.addTrack({ sessionId, trackUrl });
        logger.info("track added", { sessionId, title: track!.title, artist: track!.artist });
        await interaction.followUp(
          `Added **${track!.title}** by **${track!.artist}** to the queue.`,
        );
      },
      async (err) => {
        switch (err.code) {
          case Code.InvalidArgument:
            logger.info("add track failed: invalid url", { sessionId, trackUrl });
            await interaction.followUp(
              "Invalid URL. Please try again with a YouTube link!",
            );
            break;
          case Code.NotFound:
            logger.info("add track failed: not found", { sessionId, trackUrl });
            await interaction.followUp("That video is unavailable or the session has ended.");
            break;
          case Code.ResourceExhausted:
            logger.info("add track failed: queue full", { sessionId });
            await interaction.followUp("The queue is full!");
            break;
          default:
            logger.error("add track failed", { sessionId, err });
            await interaction.followUp("Something went wrong adding that track.");
        }
      },
    );
  } else {
    await interaction.reply("Starting playback!");
  }

  logger.info("connecting to voice channel", { sessionId, streamUrl });

  let connection: Awaited<ReturnType<typeof connectToChannel>>;
  let currentRequest: ReturnType<typeof http.get> | null = null;
  let leaving = false;

  const leave = async (message: string) => {
    if (leaving) return;
    leaving = true;
    player.removeAllListeners();
    player.stop(true);
    currentRequest?.destroy();
    connection?.destroy();
    await interaction.followUp(message).catch(() => {});
  };

  const startStream = () => {
    currentRequest?.destroy();
    const buffer = new PassThrough({ highWaterMark: MEGABYTE * BUFFER_SIZE });
    currentRequest = http.get(streamUrl, (res) => {
      res.on("end", () => {
        logger.info("stream ended, leaving voice channel", { sessionId });
        leave("Stream ended. Use /play to start a new session.");
      });
      res.pipe(buffer);
    });
    currentRequest.on("error", (err) => {
      logger.error("stream connection error, leaving voice channel", { sessionId, err });
      leave("Lost connection to the stream. Use /play to reconnect.");
    });
    buffer.once("readable", () => {
      const resource = createAudioResource(buffer, { inputType: StreamType.OggOpus });
      player.play(resource);
    });
  };

  try {
    connection = await connectToChannel(voiceChannel);
    connection.subscribe(player);
    player.removeAllListeners("error");
    player.on("error", (err) => {
      logger.error("player error, leaving voice channel", { sessionId, err });
      leave("Stream error. Use /play to reconnect.");
    });
    startStream();
    await entersState(player, AudioPlayerStatus.Playing, 5_000);
    logger.info("playback started", { sessionId, streamUrl });
  } catch (err) {
    logger.error("failed to connect or play", { sessionId, err });
  }
}

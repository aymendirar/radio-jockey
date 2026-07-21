import {
  createAudioPlayer,
  AudioPlayer,
  AudioPlayerStatus,
  createAudioResource,
  entersState,
  StreamType,
  joinVoiceChannel,
  VoiceConnection,
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
import { getSessionId, withConnectError } from "../../util/helpers.js";
import { logger } from "../../util/logger.js";
import { PassThrough } from "node:stream";
import http from "node:http";
import { hasSession, createSession, setBuffer, destroySession } from "../sessions.js";

const BUFFER_SIZE = 20 * 1024 * 1024;

const ICECAST_INTERNAL_URL = process.env.ICECAST_INTERNAL_URL!;

async function connectToChannel(channel: VoiceBasedChannel): Promise<VoiceConnection> {
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
      await radioClient.createSession({ sessionId, archive: true });
      const streamUrl = `${ICECAST_INTERNAL_URL}/stream/${sessionId}`;
      logger.info("session created", { sessionId, streamUrl });
      return streamUrl;
    },
    async (err) => {
      switch (err.code) {
        case Code.AlreadyExists: {
          await radioClient.getSession({ sessionId });
          const streamUrl = `${ICECAST_INTERNAL_URL}/stream/${sessionId}`;
          logger.info("session already exists", { sessionId, streamUrl });
          return streamUrl;
        }
        default:
          throw err;
      }
    },
  );
}

async function addTrack(
  interaction: ChatInputCommandInteraction<CacheType>,
  sessionId: string,
  trackUrl: string,
) {
  await withConnectError(
    async () => {
      const { track } = await radioClient.addTrack({ sessionId, trackUrl });
      logger.info("track added", { sessionId, title: track!.title, artist: track!.artist });
      await interaction.followUp(`Added **${track!.title}** by **${track!.artist}** to the queue.`);
    },
    async (err) => {
      switch (err.code) {
        case Code.InvalidArgument:
          logger.info("add track failed: invalid url", { sessionId, trackUrl });
          await interaction.followUp("Invalid URL. Please try again with a YouTube link!");
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
}

async function startPlayback(
  channel: VoiceBasedChannel,
  sessionId: string,
  streamUrl: string,
  interaction: ChatInputCommandInteraction<CacheType>,
  player: AudioPlayer,
): Promise<void> {
  let currentRequest: ReturnType<typeof http.get> | null = null;
  let stopped = false;

  const stop = () => {
    if (stopped) return;
    stopped = true;
    destroySession(sessionId);
    player.removeAllListeners();
    player.stop(true);
    currentRequest?.removeAllListeners();
    currentRequest?.destroy();
  };

  const leave = async (message: string) => {
    stop();
    await interaction.followUp(message).catch(() => {});
  };

  const startStream = (conn: VoiceConnection) => {
    currentRequest?.removeAllListeners();
    currentRequest?.destroy();
    const buffer = new PassThrough({ highWaterMark: BUFFER_SIZE });
    setBuffer(sessionId, buffer);
    currentRequest = http.get(streamUrl, (res) => {
      res.on("end", () => {
        logger.info("stream ended, leaving voice channel", { sessionId });
        leave("Stream ended. Use /play to start a new session.");
      });
      res.pipe(buffer);
    });
    currentRequest.on("error", (err) => {
      logger.error("stream connection error", { sessionId, err });
      leave("Lost connection to the stream. Use /play to reconnect.");
    });
    buffer.once("readable", () => {
      if (stopped) return;
      player.play(createAudioResource(buffer, { inputType: StreamType.OggOpus }));
    });
  };

  const connection = await connectToChannel(channel);
  connection.subscribe(player);
  player.on("error", (err: Error) => {
    if (stopped) return;
    logger.warn("player error, reconnecting stream", { sessionId, err });
    startStream(connection);
  });
  player.on(AudioPlayerStatus.Idle, () => {
    if (stopped) return;
    logger.info("player idle, reconnecting stream", { sessionId });
    startStream(connection);
  });
  createSession(sessionId, connection);
  startStream(connection);
  logger.info("stream started", { sessionId, streamUrl });
}

export async function handlePlayCommand(
  interaction: ChatInputCommandInteraction<CacheType>,
) {
  if (!interaction.inGuild()) {
    await interaction.reply("This command can only be used in a server!");
    return;
  }

  const member = interaction.member as GuildMember;
  if (!member.voice.channel) {
    await interaction.reply("Join a voice channel then try again!");
    return;
  }

  const trackUrl = interaction.options.getString("url");
  const sessionId = getSessionId(interaction);
  logger.info("play command received", { sessionId, trackUrl });

  if (hasSession(sessionId)) {
    if (trackUrl) {
      await interaction.reply(`Adding **${trackUrl}** to the queue...`);
      await addTrack(interaction, sessionId, trackUrl);
    } else {
      await interaction.reply("Already playing!");
    }
    return;
  }

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
    await addTrack(interaction, sessionId, trackUrl);
  } else {
    await interaction.reply("Starting playback!");
  }

  const player = createAudioPlayer();
  try {
    await startPlayback(member.voice.channel, sessionId, streamUrl, interaction, player);
  } catch (err) {
    logger.error("failed to connect or play", { sessionId, err });
    destroySession(sessionId);
    player.removeAllListeners();
    await interaction.followUp("Failed to start playback. Please try again.").catch(() => {});
  }
}

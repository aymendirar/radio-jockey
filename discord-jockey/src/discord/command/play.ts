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
import { Code, ConnectError } from "@connectrpc/connect";
import { radioClient } from "../../connect/client.js";
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
  try {
    const res = await radioClient.createSession({ sessionId });
    logger
      .withMetadata({ sessionId, streamUrl: res.streamUrl })
      .info("session created");
    return res.streamUrl;
  } catch (err) {
    if (err instanceof ConnectError && err.code === Code.AlreadyExists) {
      const res = await radioClient.getSession({ sessionId });
      logger
        .withMetadata({ sessionId, streamUrl: res.streamUrl })
        .info("session already exists");
      return res.streamUrl;
    }
    throw err;
  }
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
  logger.withMetadata({ sessionId, trackUrl }).info("play command received");

  let streamUrl: string;
  try {
    streamUrl = await getOrCreateSession(sessionId);
  } catch (err) {
    logger
      .withMetadata({ sessionId, err })
      .error("failed to get or create session");
    await interaction.reply("Failed to start a session. Please try again.");
    return;
  }

  if (trackUrl) {
    await interaction.reply(`Adding **${trackUrl}** to the queue...`);
    try {
      const { track } = await radioClient.addTrack({ sessionId, trackUrl });
      logger
        .withMetadata({ sessionId, title: track!.title, artist: track!.artist })
        .info("track added");
      await interaction.followUp(
        `Added **${track!.title}** by **${track!.artist}** to the queue.`,
      );
    } catch (err) {
      if (err instanceof ConnectError) {
        if (err.code === Code.InvalidArgument) {
          logger
            .withMetadata({ sessionId, trackUrl })
            .info("add track failed: invalid url");
          await interaction.followUp(
            "Invalid URL. Please try again with a YouTube link!",
          );
        } else if (err.code === Code.NotFound) {
          logger
            .withMetadata({ sessionId, trackUrl })
            .info("add track failed: video unavailable");
          await interaction.followUp("That video is unavailable.");
        } else if (err.code === Code.ResourceExhausted) {
          logger
            .withMetadata({ sessionId })
            .info("add track failed: queue full");
          await interaction.followUp("The queue is full!");
        } else {
          logger.withMetadata({ sessionId, err }).error("add track failed");
          await interaction.followUp("Something went wrong adding that track.");
        }
      }
      return;
    }
  } else {
    await interaction.reply("Starting playback!");
  }

  logger
    .withMetadata({ sessionId, streamUrl })
    .info("connecting to voice channel");
  const startStream = () => {
    const buffer = new PassThrough({ highWaterMark: MEGABYTE * BUFFER_SIZE });
    http.get(streamUrl, (res) => res.pipe(buffer));
    buffer.once("readable", () => {
      const resource = createAudioResource(buffer, { inputType: StreamType.OggOpus });
      player.play(resource);
    });
  };

  try {
    const connection = await connectToChannel(voiceChannel);
    connection.subscribe(player);
    player.removeAllListeners(AudioPlayerStatus.Idle);
    player.on(AudioPlayerStatus.Idle, () => {
      logger.withMetadata({ sessionId }).info("player idle, restarting stream");
      startStream();
    });
    startStream();
    await entersState(player, AudioPlayerStatus.Playing, 5_000);
    logger.withMetadata({ sessionId, streamUrl }).info("playback started");
  } catch (err) {
    logger.withMetadata({ sessionId, err }).error("failed to connect or play");
  }
}

import {
  createAudioPlayer,
  AudioPlayerStatus,
  createAudioResource,
  entersState,
  AudioPlayer,
  StreamType,
  joinVoiceChannel,
  VoiceConnectionStatus,
} from "@discordjs/voice";
import {
  ChatInputCommandInteraction,
  type CacheType,
  type VoiceBasedChannel,
} from "discord.js";

const player = createAudioPlayer();

async function playSong(player: AudioPlayer, songUrl: string) {
  const resource = createAudioResource(songUrl, {
    inputType: StreamType.Arbitrary,
  });

  player.play(resource);

  return entersState(player, AudioPlayerStatus.Playing, 5_000);
}

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

export async function registerPlayCommand(
  interaction: ChatInputCommandInteraction<CacheType>,
) {
  if (interaction.commandName === "play") {
    if (!interaction.inGuild()) {
      await interaction.reply("This command can only be used in a server!");
      return;
    }
    const guild = interaction.guild;
    if (!guild) {
      await interaction.reply("Could not access server data, try again!");
      return;
    }
    const member = await guild.members.fetch(interaction.user.id);
    const voiceChannel = member.voice.channel;
    if (!voiceChannel) {
      await interaction.reply("Join a voice channel then try again!");
      return;
    }
    try {
      await interaction.reply("Playing now!");
      const connection = await connectToChannel(voiceChannel);
      connection.subscribe(player);
      await playSong(
        player,
        "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3",
      );
    } catch (error) {
      console.error(error);
    }
  }
}

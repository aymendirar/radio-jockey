# typed: strict

class Discord
  module Command
    module Play
      sig { params(discord: Discord).void }
      def self.register(discord)
        discord.bot.command(:play) do |event|
          channel = event.user.voice_channel
          next unless channel
          event.bot.voice_connect(channel)
        end
      end
    end
  end
end

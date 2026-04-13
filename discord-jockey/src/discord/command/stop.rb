# typed: strict

class Discord
  module Command
    module Stop
      sig { params(discord: Discord).void }
      def self.register(discord)
        discord.bot.command(:stop) do |event|
          event.bot.voice_destroy(event.server)
        end
      end
    end
  end
end

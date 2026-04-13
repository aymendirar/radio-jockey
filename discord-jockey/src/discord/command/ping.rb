# typed: true

class Discord
  module Command
    module Ping
      sig { params(discord: Discord).void }
      def self.register(discord)
        discord.bot.command(:ping) do |event|
          Logger.info "received '!ping' message"
          event.respond!(content: "pong")
          event.respond!(content: event.server.id.to_s)
        end
      end
    end
  end
end

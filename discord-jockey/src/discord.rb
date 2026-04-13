# typed: true

require "discordrb"

class Discord
  sig { returns(Discordrb::Commands::CommandBot) }
  attr_reader :bot

  sig { returns(T::Hash[String, String]) }
  attr_reader :sessions

  sig { void }
  def initialize
    @bot = T.let(
      Discordrb::Commands::CommandBot.new(token: Util::Env::DISCORD_API_KEY, prefix: "!"),
      Discordrb::Commands::CommandBot
    )
    @sessions = T.let({}, T::Hash[String, String])
    register_commands
  end

  sig { void }
  def register_commands
    Command::Ping.register(self)
    Command::Play.register(self)
    Command::Stop.register(self)
  end

  sig { void }
  def run
    @bot.run
  end
end

# typed: strict

require_relative "initializers/init"
require_relative "proto/discord_jockey_services_pb"

class Main
  sig { void }
  def self.run
    Logger.info("client loaded", host: Util::Env::HOST, port: Util::Env::PORT)

    server = DiscordJockeyService::Stub.new("#{Util::Env::HOST}:#{Util::Env::PORT}",
      :this_channel_is_insecure)
    5.times do
      response = server.ping(PingRequest.new)
      pp response.message
    end

    Logger.info("initializing...")
    discord = Discord.new
    Logger.info("running...")
    discord.run
  end
end

Main.run

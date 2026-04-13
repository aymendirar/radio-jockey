# typed: true
# frozen_string_literal: true

require "sorbet-runtime"

module Util
  module Env
    HOST = T.let(ENV["HOST"], T.nilable(String))
    PORT = T.let(ENV["PORT"], T.nilable(String))
    DISCORD_API_KEY = T.let(ENV["DISCORD_API_KEY"], T.nilable(String))
  end
end

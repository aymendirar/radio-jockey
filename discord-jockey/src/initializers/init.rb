# typed: false

require "dotenv/load" if Gem.loaded_specs.key?("dotenv")
require "grpc"
require "sorbet-runtime"
require_relative "zeitwerk"
require_relative "../util/logger"

Logger.init

class Module
  include T::Sig
  include T::Helpers
end

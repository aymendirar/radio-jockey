# typed: true

require "zeitwerk"
require_relative "../util/path"

loader = Zeitwerk::Loader.new
loader.push_dir("#{Util::Path.project_root}/src")
loader.ignore("#{Util::Path.project_root}/src/initializers")
loader.ignore("#{Util::Path.project_root}/src/proto")
loader.ignore("#{Util::Path.project_root}/src/main.rb")
loader.setup

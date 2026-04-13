# typed: true

require "logger"

class Logger
  def self.init
    @logger = ::Logger.new($stdout)
    @logger.progname = "discord-client"
    @logger.formatter = proc do |severity, datetime, progname, msg|
      "[#{progname}] #{severity}: #{msg}\n"
    end
    $stdout.sync = true
    $stderr.sync = true
  end

  def self.info(msg, **attrs)
    @logger.info(format(msg, attrs))
  end

  def self.warn(msg, **attrs)
    @logger.warn(format(msg, attrs))
  end

  def self.error(msg, **attrs)
    @logger.error(format(msg, attrs))
  end

  def self.format(msg, attrs)
    return msg if attrs.empty?

    formatted = attrs.each_with_index.map do |(k, v), i|
      i < attrs.size - 1 ? "#{k}: #{v}," : "#{k}: #{v}"
    end.join(" ")

    "#{msg} { #{formatted} }"
  end
  private_class_method :format
end

module Util
  class Path
    def self.project_root
      File.expand_path("../..", __dir__)
    end

    def self.repo_root
      File.expand_path("../../..", __dir__)
    end
  end
end

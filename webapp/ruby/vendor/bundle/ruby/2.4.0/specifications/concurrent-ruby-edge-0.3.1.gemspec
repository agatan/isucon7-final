# -*- encoding: utf-8 -*-
# stub: concurrent-ruby-edge 0.3.1 ruby lib

Gem::Specification.new do |s|
  s.name = "concurrent-ruby-edge".freeze
  s.version = "0.3.1"

  s.required_rubygems_version = Gem::Requirement.new(">= 0".freeze) if s.respond_to? :required_rubygems_version=
  s.require_paths = ["lib".freeze]
  s.authors = ["Jerry D'Antonio".freeze, "Petr Chalupa".freeze, "The Ruby Concurrency Team".freeze]
  s.date = "2017-02-26"
  s.description = "These features are under active development and may change frequently. They are expected not to\nkeep backward compatibility (there may also lack tests and documentation). Semantic versions will\nbe obeyed though. Features developed in `concurrent-ruby-edge` are expected to move to `concurrent-ruby` when final.\nPlease see http://concurrent-ruby.com for more information.\n".freeze
  s.email = "concurrent-ruby@googlegroups.com".freeze
  s.extra_rdoc_files = ["README.md".freeze, "LICENSE.txt".freeze]
  s.files = ["LICENSE.txt".freeze, "README.md".freeze]
  s.homepage = "http://www.concurrent-ruby.com".freeze
  s.licenses = ["MIT".freeze]
  s.required_ruby_version = Gem::Requirement.new(">= 1.9.3".freeze)
  s.rubygems_version = "2.6.13".freeze
  s.summary = "Edge features and additions to the concurrent-ruby gem.".freeze

  s.installed_by_version = "2.6.13" if s.respond_to? :installed_by_version

  if s.respond_to? :specification_version then
    s.specification_version = 4

    if Gem::Version.new(Gem::VERSION) >= Gem::Version.new('1.2.0') then
      s.add_runtime_dependency(%q<concurrent-ruby>.freeze, ["= 1.0.5"])
    else
      s.add_dependency(%q<concurrent-ruby>.freeze, ["= 1.0.5"])
    end
  else
    s.add_dependency(%q<concurrent-ruby>.freeze, ["= 1.0.5"])
  end
end

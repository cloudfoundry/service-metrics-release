desc 'run all the specs'
task spec: %w(spec:system spec:template)

namespace :spec do
  require 'rspec/core/rake_task'

  desc 'run all of the system tests'
  RSpec::Core::RakeTask.new(:system) do |t|
    t.pattern = FileList['spec/system/**/*_spec.rb']
  end

  desc 'run all of the template tests'
  RSpec::Core::RakeTask.new(:template) do |t|
    t.pattern = FileList['spec/templates/**/*_spec.rb']
  end
end

task default: :spec

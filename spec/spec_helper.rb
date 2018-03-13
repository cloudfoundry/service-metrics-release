ROOT = File.expand_path('..', __dir__)

module Helpers
  module Environment
    def cf_api
      ENV['CF_API'] || 'https://api.bosh-lite.com'
    end

    def cf_username
      ENV['CF_USERNAME'] || "admin"
    end

    def cf_password
      ENV['CF_PASSWORD'] || "admin"
    end

    def doppler_address
      ENV['DOPPLER_ADDR'] || "wss://doppler.bosh-lite.com:443"
    end

    def deployment_name
      ENV.fetch('DEPLOYMENT_NAME')
    end

    def origin
      ENV.fetch('ORIGIN')
    end

    def target_cf
      if ENV['CF_SSL_VALIDATION']
        `cf api #{cf_api}`
      else
        `cf api --skip-ssl-validation #{cf_api}`
      end
    end

    def cf_login
      `cf auth #{cf_username} #{cf_password}`
    end

    def cf_auth_token
      target_cf
      cf_login
      `cf oauth-token | tail -n 1`.strip!
    end
  end
end

RSpec.configure do |config|
  config.include Helpers::Environment
  config.filter_run :focus
  config.run_all_when_everything_filtered = true
  config.order = 'random'
  config.full_backtrace = true
  config.color = true
end

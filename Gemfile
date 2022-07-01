source 'https://rubygems.org'

git_source(:github) do |repo_name|
  repo_name = "#{repo_name}/#{repo_name}" unless repo_name.include?("/")
  "https://github.com/#{repo_name}.git"
end


gem 'rails', '~> 6.1.6'

gem 'jbuilder'
gem 'jquery-rails'
gem 'pg'
gem 'puma'
gem 'sass-rails'
gem 'turbolinks'
gem 'webpacker'

gem 'bootsnap', require: false
gem 'bootstrap-sass', '~> 3.4.1'
gem 'devise', '~> 4.7.1'
gem 'dropzonejs-rails', '~> 0.7.3'
gem 'font-awesome-sass', '~> 4.7.0'
gem 'kaminari', '~> 1.2.1'

# Locking these for security updates:
gem 'loofah', '~> 2.3.1'
gem 'mimemagic', '~> 0.3.10'
gem 'nokogiri', '~> 1.10.10'
gem 'paperclip', '~> 6.1.0'
gem 'rack', '~> 2.1.4'
gem 'sprockets', '~> 3.7.2'
gem 'websocket-extensions', '~> 0.1.5'

group :development, :test do
  # Call 'byebug' anywhere in the code to stop execution and get a debugger console
  gem 'byebug', platform: :mri
  gem 'rspec-rails', '~> 3.5'
end

group :development do
  # Access an IRB console on exception pages or by using <%= console %> anywhere in the code.
  gem 'web-console', '>= 3.3.0'
  gem 'listen', '~> 3.0.5'
  # Spring speeds up development by keeping your application running in the background. Read more: https://github.com/rails/spring
  gem 'spring'
  gem 'spring-watcher-listen', '~> 2.0.0'
end

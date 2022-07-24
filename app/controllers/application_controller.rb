class ApplicationController < ActionController::Base
  before_action :check_user_setup

  # HACK: Override this for any non-GET/POST requests to force a GET redirect.
  def redirect_to(options = {}, response_options = {})
    response_options.merge!(status: :see_other) unless ["GET", "POST"].include?(request.request_method)
    super(options, response_options)
  end

  private

  def not_authenticated
    redirect_to login_path
  end

  def check_user_setup
    if User.count.zero? && (request.fullpath != setup_path)
      redirect_to setup_path
    end
  end
end

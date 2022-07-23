class ApplicationController < ActionController::Base
  before_action :check_user_setup

  protected

  # HACK: Use this on any non-GET requests instead of `redirect_to` to force a GET.
  def redirect_get(options = {}, response_options = {})
    redirect_to(options, response_options.merge(status: :see_other))
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

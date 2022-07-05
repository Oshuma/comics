class ApplicationController < ActionController::Base
  before_action :check_user_setup

  private

  def not_authenticated
    redirect_to(login_path, alert: 'Login required.')
  end

  def check_user_setup
    # if User.count.zero? && (request.full_path != setup_path)
    #   redirect_to setup_path
    # end
  end
end

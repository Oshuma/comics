class ApplicationController < ActionController::Base
  # Prevent CSRF attacks by raising an exception.
  # For APIs, you may want to use :null_session instead.
  protect_from_forgery with: :exception

  before_action :check_user_setup

  protected

  def authenticate_admin!
    redirect_to new_user_session_path unless current_user && current_user.admin?
  end

  private

  def check_user_setup
    if User.count.zero? && (request.fullpath != setup_path)
      redirect_to setup_path
    end
  end
end

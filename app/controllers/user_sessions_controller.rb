class UserSessionsController < ApplicationController
  before_action :require_login, only: [:destroy]

  def create
    @user = login(params[:email], params[:password], params[:remember])

    if @user
      redirect_back_or_to(root_path, notice: 'Login successful.')
    else
      flash.now[:alert] = 'Login failed.'
      render action: 'new'
    end
  end

  def destroy
    logout
    redirect_to(login_path, notice: 'Logged out.')
  end
end

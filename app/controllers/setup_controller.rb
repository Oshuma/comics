class SetupController < ApplicationController

  def new
    @user = User.new
  end

  def create
    @user = User.new(user_params)
    @user.admin = true

    if @user.save
      sign_in(:user, @user)
      redirect_to root_path, notice: 'Admin user created.'
    else
      render :new
    end
  end

  private

  def user_params
    params.require(:user).permit(
      :email,
      :password,
      :password_confirmation
    )
  end

end

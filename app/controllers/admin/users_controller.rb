class Admin::UsersController < ApplicationController

  before_action :authenticate_admin!

  def create
    @user = User.new(user_params)

    if @user.save
      redirect_to admin_path, notice: 'User added.'
    else
      redirect_to admin_path, alert: 'There was a problem adding that user.'
    end
  end

  def enable_admin
    @user = User.find(params[:id])

    if @user.update(admin: true)
      redirect_to admin_path, notice: 'User is now an admin.'
    else
      redirect_to admin_path, alert: 'Could not make user an admin.'
    end
  end

  def disable_admin
    @user = User.find(params[:id])

    if @user == current_user
      return redirect_to admin_path, alert: "Can't remove yourself as admin."
    end

    if @user.update(admin: false)
      redirect_to admin_path, notice: 'User is no longer an admin.'
    else
      redirect_to admin_path, alert: 'Could not disable admin user.'
    end
  end

  def destroy
    @user = User.find(params[:id])

    if @user == current_user
      return redirect_to admin_path, alert: "Can't delete your own account."
    end

    if @user.destroy
      redirect_to admin_path, notice: 'User and their comics were removed.'
    else
      redirect_to admin_path, alert: 'Could not remove user.'
    end
  end

  private

  def user_params
    params.require(:user).permit(
      :email,
      :password,
      :password_confirmation,
      :admin,
    )
  end

end

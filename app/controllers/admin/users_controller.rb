class Admin::UsersController < ApplicationController

  before_action :authenticate_admin!

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

end

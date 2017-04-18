class AdminController < ApplicationController

  before_action :authenticate_admin!

  def index
    @users = User.order(created_at: :asc).page(params[:page])
  end

end

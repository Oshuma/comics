class HistoryController < ApplicationController

  before_action :authenticate_user!

  def index
    @histories = current_user.histories.order(created_at: :desc).page(params[:page])
  end

end

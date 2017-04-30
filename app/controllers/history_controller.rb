class HistoryController < ApplicationController

  before_action :authenticate_user!

  def index
    @histories = current_user.histories.page(params[:page])
  end

end

class StatsController < ApplicationController

  before_action :authenticate_user!

  def index
    @groups = current_user.groups.order(name: :asc)
    @total_size = @groups.sum(&:disk_size)
  end

end

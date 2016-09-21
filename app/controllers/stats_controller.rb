class StatsController < ApplicationController

  def index
    @groups = current_user.groups.order(name: :asc)
    @total_size = @groups.sum(&:disk_size)
  end

end

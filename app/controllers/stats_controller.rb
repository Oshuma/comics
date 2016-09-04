class StatsController < ApplicationController

  def index
    @groups = current_user.groups.order(name: :asc)
  end

end

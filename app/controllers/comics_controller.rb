class ComicsController < ApplicationController
  before_action :require_login

  def new
    @groups = current_user.groups.order(name: :asc)
  end
end

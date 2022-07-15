class GroupsController < ApplicationController
  before_action :require_login

  def index
    @groups = current_user.groups.order(name: :asc)
  end

  def create
    @group = current_user.groups.build(group_params)

    if @group.save
      redirect_to new_comic_path(group_id: @group.id), notice: 'Group added.'
    else
      redirect_to :back, alert: 'There was a problem adding that group.'
    end
  end

  private

  def group_params
    params.require(:group).permit(:name)
  end
end

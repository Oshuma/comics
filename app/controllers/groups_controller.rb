class GroupsController < ApplicationController
  before_action :require_login

  def index
    @groups = current_user.groups.order(name: :asc)
  end

  def show
    @group = current_user.groups.find(params[:id])
    @groups = current_user.groups.order(name: :asc)
    @comics = @group.comics
  end

  def create
    @group = current_user.groups.build(group_params)

    if @group.save
      redirect_to new_comic_path(group_id: @group.id), notice: 'Group added.'
    else
      redirect_to :back, alert: 'There was a problem adding that group.'
    end
  end

  def update
    @group = current_user.groups.find(params[:id])
    if @group.update(group_params)
      redirect_to groups_path, notice: 'Group saved.'
    else
      redirect_to groups_path, alert: 'There was a problem saving that group.'
    end
  end

  def destroy
    @group = current_user.groups.find(params[:id])
    @group.destroy
    redirect_get groups_path, notice: 'Group deleted.'
  end

  private

  def group_params
    params.require(:group).permit(:name)
  end
end

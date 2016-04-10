class GroupsController < ApplicationController

  def index
    @groups = current_user.groups.order(name: :asc)
  end

  def show
    @group = current_user.groups.find(params[:id])
    @comics = @group.comics
  end

  def create
    @group = current_user.groups.build(group_params)

    if @group.save
      redirect_to :back, notice: 'Group added.'
    else
      redirect_to :back, alert: 'There was a problem adding that group.'
    end
  end

  def destroy
    @group = current_user.groups.find(params[:id])
    @group.destroy
    redirect_to root_path, notice: 'Group removed.'
  end

  private

  def group_params
    params.require(:group).permit(:name)
  end

end

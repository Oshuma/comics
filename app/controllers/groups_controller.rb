class GroupsController < ApplicationController

  def index
    @groups = current_user.groups.order(name: :asc)
  end

  def show
    @groups = current_user.groups.order(name: :asc)
    @group = current_user.groups.find(params[:id])
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
      redirect_to :back, notice: 'Group updated.'
    else
      redirect_to :back, alert: 'There was a problem updating that group.'
    end
  end

  def destroy
    @group = current_user.groups.find(params[:id])
    @group.destroy

    respond_to do |format|
      format.html { redirect_to root_path, notice: 'Group removed.' }
      format.js
    end
  end

  def delete_comics
    @group = current_user.groups.find(params[:id])
    @group.delete_comics!

    redirect_to root_path
  end

  private

  def group_params
    params.require(:group).permit(:name)
  end

end

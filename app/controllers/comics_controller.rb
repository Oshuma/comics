class ComicsController < ApplicationController
  before_action :require_login

  def new
    @groups = current_user.groups.order(name: :asc)
    @comic = current_user.comics.build
  end

  def create
    @comic = current_user.comics.build(comic_params)
    if @comic.save
      redirect_to new_comic_path, notice: 'Comic uploaded and is now processing!'
    else
      redirect_to new_comic_path, alert: 'There was an issue uploading that comic.'
    end
  end

  def destroy
    @comic = current_user.comics.find(params[:id])
    @comic.destroy
    redirect_to group_path(@comic.group), status: :see_other, notice: 'Comic deleted.'
  end

  private

  def comic_params
    params.require(:comic).permit(:archive, :group_id)
  end
end

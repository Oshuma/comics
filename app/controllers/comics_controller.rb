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

  private

  def comic_params
    params.require(:comic).permit(:archive, :group_id)
  end
end

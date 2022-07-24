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
    redirect_to group_path(@comic.group), notice: 'Comic deleted.'
  end

  def read
    @comic = current_user.comics.find(params[:id])

    if @first_unread_page = @comic.pages.first_unread
      redirect_to comic_page_path(@comic, @first_unread_page)
    else
      redirect_to comic_path(@comic)
    end
  end

  def finish
    @comic = current_user.comics.find(params[:id])
    @comic.finished!

    redirect_to group_path(@comic.group)
  end

  def delete_read
    DeleteReadComicsJob.perform_later(current_user)
    redirect_to root_path
  end

  private

  def comic_params
    params.require(:comic).permit(:archive, :group_id)
  end
end

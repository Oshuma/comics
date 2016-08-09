class ComicsController < ApplicationController

  def new
    @groups = current_user.groups.order(name: :asc)
  end

  def create
    params[:comics].each do |comic_params|
      # TODO: Move this to a worker.
      comic = current_user.comics.build
      next if comic.import_from_archive(comic_params)

      # If there's a problem importing, redirect to the uploader.
      redirect_to new_comic_path
    end

    respond_to do |format|
      format.json { render json: @comics }
    end
  end

  def show
    @comic = current_user.comics.find(params[:id])
  end

  def read
    @comic = current_user.comics.find(params[:id])

    if @first_unread_page = @comic.pages.first_unread
      redirect_to comic_page_path(@comic, @first_unread_page)
    else
      redirect_to comic_path(@comic)
    end
  end

  def destroy
    @comic = current_user.comics.find(params[:id])
    @comic.destroy
    redirect_to root_path, notice: 'Comic deleted.'
  end

end

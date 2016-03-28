class ComicsController < ApplicationController

  def index
    @comic = Comic.order(filename: :asc)
  end

  def new
    @comic = Comic.new
  end

  def create
    @comic = Comic.new

    params[:comics].each do |comic_params|
      # TODO: Move this to a worker.
      comic = Comic.new
      next if comic.import_from_archive(comic_params)

      # If there's a problem importing, just render :new
      render :new
    end

    redirect_to root_path, notice: 'Comics imported.'
  end

  def show
    @comic = Comic.find(params[:id])
  end

  def read
    @comic = Comic.find(params[:id])

    if @first_unread_page = @comic.pages.first_unread
      redirect_to comic_page_path(@comic, @first_unread_page)
    else
      redirect_to comic_path(@comic)
    end
  end

  def destroy
    @comic = Comic.find(params[:id])
    @comic.destroy
    redirect_to comics_path, notice: 'Comic deleted.'
  end

end

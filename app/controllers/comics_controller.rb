class ComicsController < ApplicationController

  def index
    @comic = Comic.all
  end

  def new
    @comic = Comic.new
  end

  def create
    @comic = Comic.new

    if @comic.import_from_archive(comic_params)
      redirect_to root_path, notice: 'Comic imported.'
    else
      render :new
    end
  end

  def show
    @comic = Comic.find(params[:id])

    if @first_unread_page = @comic.pages.first_unread
      redirect_to comic_page_path(@comic, @first_unread_page)
    end
  end

  private

  def comic_params
    params.require(:comic).permit(
      :title,
      :issue,
      :cover_date,
      :archive,
    )
  end

end

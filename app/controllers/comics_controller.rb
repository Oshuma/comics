class ComicsController < ApplicationController

  def index
    @comic = Comic.all
  end

  def new
    @comic = Comic.new
    3.times { @comic.pages.build }
  end

  def create
    @comic = Comic.new(comic_params)

    if @comic.save
      redirect_to comics_path, notice: 'Comic saved.'
    else
      render :new
    end
  end

  def show
    @comic = Comic.find(params[:id])
  end

  private

  def comic_params
    params.require(:comic).permit(
      :title,
      :issue,
      :cover_date,

      pages_attributes: [
        :number,
        :image
      ]
    )
  end

end

class PagesController < ApplicationController

  def show
    @comic = Comic.find(params[:comic_id])
    @page = @comic.pages.find(params[:id])
  end

end

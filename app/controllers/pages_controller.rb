class PagesController < ApplicationController

  before_action :set_page

  def show
    @page.set_as_current!
  end

  # PUT next_page
  def next_page
    @next_page = @page.next!

    if @next_page
      redirect_to comic_page_path(@comic, @next_page)
    else
      redirect_to next_comic
    end
  end

  # PUT previous_page
  def previous_page
    @previous_page = @page.previous!

    if @previous_page
      redirect_to comic_page_path(@comic, @previous_page)
    else
      redirect_to @comic.group
    end
  end

  private

  # Returns next Comic or current Group.
  def next_comic
    result = @comic.group.next_comic(@comic)
    result.is_a?(Comic) ? read_comic_path(result) : result
  end

  def set_page
    @comic = current_user.comics.find(params[:comic_id])
    @page = @comic.pages.find(params[:id])
  end

end

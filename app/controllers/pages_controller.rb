class PagesController < ApplicationController
  before_action :require_login
  before_action :set_page

  def show
    @page.set_as_current!
  end

  def next_page
    @next_page = @page.next!

    if @next_page
      redirect_get comic_page_path(@comic, @next_page)
    else
      # @comic.record_history!
      redirect_get next_comic
    end
  end

  def previous_page
    @previous_page = @page.previous!

    if @previous_page
      redirect_get comic_page_path(@comic, @previous_page)
    else
      redirect_get @comic.group
    end
  end

  private

  def next_comic
    result = @comic.group.next_comic(@comic)
    result.is_a?(Comic) ? read_comic_path(result) : result
  end

  def set_page
    @comic = current_user.comics.find(params[:comic_id])
    @page = @comic.pages.find(params[:id])
  end
end

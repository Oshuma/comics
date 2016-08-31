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

  def finish
    @comic = current_user.comics.find(params[:id])
    @comic.finished!

    redirect_to group_path(@comic.group)
  end

  def move
    @comic = current_user.comics.find(params[:id])
    @group = current_user.groups.find(params[:group_id])
    @comic.move_to(@group)

    respond_to do |format|
      format.html { redirect_to(:back, notice: 'Comic moved.') }
      format.js
    end
  end

  def destroy
    @comic = current_user.comics.find(params[:id])
    @comic.destroy

    respond_to do |format|
      format.html { redirect_to root_path, notice: 'Comic deleted.' }
      format.js
    end
  end

  def delete_read
    current_user.comics.each { |comic| comic.destroy if comic.read? }
    redirect_to root_path
  end

end

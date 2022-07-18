class CreatePagesJob < ApplicationJob
  queue_as :default

  def perform(comic)
    comic.create_pages_from_archive!
  end
end

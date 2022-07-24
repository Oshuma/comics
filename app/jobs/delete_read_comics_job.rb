class DeleteReadComicsJob < ApplicationJob
  queue_as :default

  def perform(user_or_group)
    user_or_group.comics.each { |c| c.destroy if c.read? }
  end
end

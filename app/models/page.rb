class Page < ApplicationRecord
  has_one_attached :image do |attachable|
    attachable.variant :thumb, resize_to_limit: [300, 300]
  end

  belongs_to :comic

  # Note: This needs to be a class method (not scope),
  # so it doesn't return an AR relation.
  def self.first_unread
    where(read: false).first
  end

  def previous_page
    comic.pages.where(number: (number - 1)).first
  end

  # Marks this page as unread and returns the previous page or false.
  def previous!
    if previous_page
      previous_page.update(read: false)
      comic.pages.where('number > ?', number).update_all(read: false)

      previous_page
    else
      false
    end
  end

  def next_page
    comic.pages.where(number: (number + 1)).first
  end

  # Marks this page as read and returns the next page or false.
  def next!
    update(read: true) ? next_page : false
  end

  # Marks this page and all next ones as unread; mark all previous pages as read.
  def set_as_current!
    comic.pages.where('number < ?', number).update_all(read: true)
    comic.pages.where('number > ?', number).update_all(read: false)
    update(read: false)
  end
end

class Page < ApplicationRecord

  belongs_to :comic

  has_attached_file :image,
    styles: {
    thumb: '300x300>'
  }

  validates_attachment :image,
    presence: true,
    content_type: { content_type: ['image/jpg', 'image/jpeg', 'image/gif', 'image/png'] }


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

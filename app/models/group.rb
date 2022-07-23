class Group < ApplicationRecord
  belongs_to :user
  has_many :comics, dependent: :destroy

  validates :name, presence: true

  def cover_image_thumb
    @cover_image_thumb ||= comics.first.try(:cover_image_thumb)
  end

  def delete_comics!
    # comics.destroy_all
  end

  def delete_read_comics!
    # comics.each { |comic| comic.destroy if comic.read? }
  end

  def disk_size
    # comics.map { |comic| comic.pages.map { |page| page.image_file_size }.sum }.sum
  end

  def has_comics?
    comics.present?
  end

  # Returns the Comic after `current_comic` or `self` if last Comic.
  def next_comic(current_comic)
    comics.each_slice(2) do |one, two|
      if current_comic == one
        return two if two.present?
      end
    end

    self
  end

  def read?
    comics.all? { |comic| comic.read? }
  end

  def reading?
    comics.any? { |comic| comic.reading? }
  end

  def processing?
    comics.any? { |comic| comic.processing? }
  end
end

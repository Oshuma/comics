class Group
  include Mongoid::Document
  include Mongoid::Timestamps

  belongs_to :user
  has_many :comics, dependent: :destroy, order: { filename: :asc }

  field :name, type: String

  validates :name, presence: true


  def delete_comics!
    comics.destroy_all
  end

  def disk_size
    comics.map { |comic| comic.pages.map { |page| page.image_file_size }.sum }.sum
  end

  def has_comics?
    comics.present?
  end

  # Returns the Comic after `current_comic` or `self` if last Comic.
  def next_comic(current_comic)
    # FIXME: This is lame, but so is mongo. Make this a proper query when we're using a decent db.
    comics.each_slice(2) do |one, two|
      return two if current_comic.id.to_s == one.id.to_s
    end

    self
  end

  def read?
    comics.all? { |comic| comic.read? }
  end

  def reading?
    comics.any? { |comic| comic.reading? }
  end
end

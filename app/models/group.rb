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

  def read?
    comics.all? { |comic| comic.read? }
  end

  def reading?
    comics.any? { |comic| comic.reading? }
  end
end

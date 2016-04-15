class Group
  include Mongoid::Document
  include Mongoid::Timestamps

  belongs_to :user
  has_many :comics, dependent: :destroy, order: { filename: :asc }

  field :name, type: String

  validates :name, presence: true

  def has_comics?
    comics.present?
  end

  def read?
    comics.all? { |comic| comic.read? }
  end
end

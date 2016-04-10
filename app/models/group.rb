class Group
  include Mongoid::Document
  include Mongoid::Timestamps

  belongs_to :user
  has_many :comics, dependent: :destroy, order: { filename: :asc }

  field :name, type: String

  validates :name, presence: true

  def read?
    comics.all? { |comic| comic.read? }
  end
end

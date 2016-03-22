class Comic
  include Mongoid::Document
  include Mongoid::Timestamps

  has_many :pages
  accepts_nested_attributes_for :pages

  field :title, type: String
  field :issue, type: String
  field :cover_date, type: Date

  validates :title, presence: true
  validates :issue, presence: true
end

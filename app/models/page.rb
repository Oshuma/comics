class Page
  include Mongoid::Document
  include Mongoid::Timestamps
  include Mongoid::Paperclip

  belongs_to :comic

  field :number, type: Integer
  field :read, type: Mongoid::Boolean

  has_mongoid_attached_file :image

  validates :number, presence: true, numericality: { only_integer: true }

  validates_attachment :image,
    presence: true,
    content_type: { content_type: ['image/jpg', 'image/jpeg', 'image/gif', 'image/png'] }

end

class Page
  include Mongoid::Document
  include Mongoid::Timestamps
  include Mongoid::Paperclip

  belongs_to :comic

  field :number, type: Integer
  field :read, type: Mongoid::Boolean, default: false

  has_mongoid_attached_file :image, styles: { thumb: '200x200>' }

  validates :number, presence: true, numericality: { only_integer: true }

  validates_attachment :image,
    presence: true,
    content_type: { content_type: ['image/jpg', 'image/jpeg', 'image/gif', 'image/png'] }

end

class History < ApplicationRecord
  belongs_to :user

  # These are stored as strings, since the Group/Comic records might get deleted by the User at some point.
  validates :group_name, presence: true
  validates :comic_name, presence: true
end

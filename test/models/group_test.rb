require "test_helper"

class GroupTest < ActiveSupport::TestCase
  test "has comics association" do
    assert groups(:one).respond_to?(:comics)
  end
end

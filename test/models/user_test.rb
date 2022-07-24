require "test_helper"

class UserTest < ActiveSupport::TestCase
  test "has comics association" do
    assert users(:one).respond_to?(:comics)
  end
end

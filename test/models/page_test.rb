require "test_helper"

class PageTest < ActiveSupport::TestCase
  test ".first_unread" do
    assert Page.respond_to?(:first_unread)
  end
end

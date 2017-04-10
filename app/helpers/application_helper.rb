module ApplicationHelper

  def thumb_or_placeholder_image_tag(image)
    image_tag(image.exists? ? image.url(:thumb) : 'placeholder.png')
  end

end

class Comic < ApplicationRecord
  has_one_attached :archive

  belongs_to :user
  belongs_to :group
  has_many :pages, -> { order(number: :asc) }, dependent: :destroy, autosave: true

  def cover_image_thumb
    # @cover_image_thumb ||= pages.first.try(:image)
  end

  def upload(uploaded_file)
    # self.filename = uploaded_file.original_filename

    # if save
    #   create_pages_from_archive(uploaded_file)
    # else
    #   false
    # end
  end

  def finished!
    # pages.update_all(read: true) ? record_history! : false
  end

  def read?
    # pages.all? { |page| page.read? }
  end

  # Returns true if there is at least one read page.
  def reading?
    # pages.any? { |page| page.read? }
  end

  def record_history!
    # user.histories.find_or_create_by!(group_name: group.name, comic_name: filename)
  end

  def move_to(group)
    # self.group = group
    # save
  end

  def pretty_filename
    # File.basename(filename, File.extname(filename))
  end

  private

  def create_pages_from_archive(archive_file)
    # tempfile = archive_file.tempfile
    # case File.extname(tempfile).downcase
    # when '.cbr'
    #   create_from_cbr(tempfile)
    # when '.cbz'
    #   create_from_cbz(tempfile)
    # else
    #   raise StandardError.new("Unknown file extension: #{tempfile}")
    # end
  end

  def create_from_cbr(tempfile)
    # raise StandardError.new("unrar not found") if `which unrar`.chomp.empty?

    # inside_temp_dir(tempfile) do
    #   `unrar e #{tempfile.path}`
    # end
  end

  def create_from_cbz(tempfile)
    # raise StandardError.new("unzip not found") if `which unzip`.chomp.empty?

    # inside_temp_dir(tempfile) do
    #   `unzip -j #{tempfile.path}`
    # end
  end

  def inside_temp_dir(tempfile, &block)
    # temp_dir = Rails.root.join('tmp', File.basename(tempfile.path))
    # Dir.mkdir(temp_dir)

    # Dir.chdir(temp_dir) do
    #   yield

    #   page_number = 1

    #   Dir['*'].sort.each do |image|
    #     begin
    #       pages.create!(number: page_number, image: File.open(image))
    #       page_number += 1
    #     # This is mainly here in case the archive has some dumbass 'Thumbs.db' file that isn't an image.
    #     rescue => e
    #       Rails.logger.warn "-- WARN: #{Time.now}: #{self.class}#inside_temp_dir: #{e.inspect}"
    #       Rails.logger.warn "-- WARN: #{Time.now}: #{self.class}#inside_temp_dir: image: #{image.inspect}"
    #       next
    #     end
    #   end
    # end

    # FileUtils.rm_rf(temp_dir)
  end
end

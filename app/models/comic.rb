class Comic < ApplicationRecord
  has_one_attached :archive

  belongs_to :user
  belongs_to :group
  has_many :pages, -> { order(number: :asc) }, dependent: :destroy, autosave: true

  after_create_commit do
    CreatePagesJob.perform_later(self)
    update(name: pretty_filename)
  end

  def cover_image_thumb
    @cover_image_thumb ||= pages.first.try(:image).try(:variant, :thumb)
  end

  def finished!
    pages.update_all(read: true) ? record_history! : false
  end

  def read?
    pages.all? { |page| page.read? }
  end

  # Returns true if there is at least one read page.
  def reading?
    pages.any? { |page| page.read? }
  end

  # TODO: Implement.
  def record_history!
    # user.histories.find_or_create_by!(group_name: group.name, comic_name: filename)
  end

  # def move_to(group)
  #   # self.group = group
  #   # save
  # end

  def pretty_filename
    archive.filename.base
  end

  def create_pages_from_archive!
    update(processing: true)

    begin
      case File.extname(archive.filename.to_s).downcase
      when '.cbr'
        create_from_cbr
      when '.cbz'
        create_from_cbz
      else
        raise StandardError.new("Unknown file extension: #{archive.filename}")
      end
    ensure
      update(processing: false)
    end
  end

  private

  def create_from_cbr
    raise StandardError.new("unrar not found") if `which unrar`.chomp.empty?

    inside_temp_dir do
      archive.open do |f|
        `unrar e #{f.path}`
      end
    end
  end

  def create_from_cbz
    raise StandardError.new("unzip not found") if `which unzip`.chomp.empty?

    inside_temp_dir do
      archive.open do |f|
        `unzip -j #{f.path}`
      end
    end
  end

  def inside_temp_dir(&block)
    temp_dir = File.join(Dir.tmpdir, archive.filename.to_s)
    FileUtils.remove_entry_secure(temp_dir) if Dir.exist?(temp_dir)
    Dir.mkdir(temp_dir)

    Dir.chdir(temp_dir) do
      yield

      page_number = 1

      Dir['*'].sort.each do |image|
        begin
          page = pages.build(number: page_number)
          page.image.attach(io: File.open(image), filename: image)
          page.save!

          page.save!
          page_number += 1
        rescue => e
          # This is mainly here in case the archive has some dumbass 'Thumbs.db' file that isn't an image.
          Rails.logger.warn "-- WARN: #{Time.now}: #{self.class}#create_from_cbr: #{e.inspect}"
          Rails.logger.warn "-- WARN: #{Time.now}: #{self.class}#create_from_cbr: image: #{image.inspect}"
          next
        end
      end
    end

    FileUtils.remove_entry_secure(temp_dir) if Dir.exist?(temp_dir)
  end
end

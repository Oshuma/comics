class Comic
  include Mongoid::Document
  include Mongoid::Timestamps

  belongs_to :user
  has_many :pages, dependent: :destroy, autosave: true, order: { number: :asc }

  belongs_to :group

  field :filename, type: String

  validates :filename, presence: true


  def import_from_archive(comic_params)
    self.filename = comic_params[:archive].original_filename
    self.group_id = comic_params[:group_id]

    if save
      create_pages_from_archive(comic_params[:archive])
    else
      false
    end
  end

  def read?
    pages.all? { |page| page.read? }
  end

  # Returns true if there is at least one read page.
  def reading?
    pages.any? { |page| page.read? }
  end

  private

  def create_pages_from_archive(archive_file)
    tempfile = archive_file.tempfile
    case File.extname(tempfile).downcase
    when '.cbr'
      create_from_cbr(tempfile)
    when '.cbz'
      create_from_cbz(tempfile)
    else
      raise StandardError.new("Unknown file extension: #{tempfile}")
    end
  end

  def create_from_cbr(tempfile)
    raise StandardError.new("unrar not found") if `which unrar`.chomp.empty?

    inside_temp_dir(tempfile) do
      `unrar e #{tempfile.path}`
    end
  end

  def create_from_cbz(tempfile)
    raise StandardError.new("unzip not found") if `which unzip`.chomp.empty?

    inside_temp_dir(tempfile) do
      `unzip -j #{tempfile.path}`
    end
  end

  def inside_temp_dir(tempfile, &block)
    temp_dir = Rails.root.join('tmp', File.basename(tempfile.path))
    Dir.mkdir(temp_dir)

    Dir.chdir(temp_dir) do
      yield

      page_number = 1

      Dir['*'].sort.each do |image|
        begin
          pages.create!(number: page_number, image: File.open(image))
          page_number += 1
        rescue Mongoid::Errors::Validations
          # This is mainly here in case the archive has some dumbass 'Thumbs.db' file that isn't an image.
          next
        end
      end
    end

    FileUtils.rm_rf(temp_dir)
  end
end

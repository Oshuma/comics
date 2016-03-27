class Comic
  include Mongoid::Document
  include Mongoid::Timestamps

  has_many :pages, dependent: :destroy, autosave: true, order: { number: :asc }

  field :filename, type: String

  validates :filename, presence: true


  def import_from_archive(comic_params)
    self.filename = comic_params[:archive].original_filename

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
    case File.extname(tempfile)
    when '.cbr'
      create_from_cbr(tempfile)
    else
      raise StandardError.new("Unknown file extension: #{tempfile}")
    end
  end

  def create_from_cbr(tempfile)
    raise StandardError.new("unrar not found") if `which unrar`.chomp.empty?

    temp_dir = Rails.root.join('tmp', File.basename(tempfile.path))
    Dir.mkdir(temp_dir)

    Dir.chdir(temp_dir) do
      `unrar e #{tempfile.path}`

      page_number = 1

      Dir['*'].sort.each do |image|
        pages.create!(number: page_number, image: File.open(image))
        page_number += 1
      end
    end

    FileUtils.rm_rf(temp_dir)
  end
end

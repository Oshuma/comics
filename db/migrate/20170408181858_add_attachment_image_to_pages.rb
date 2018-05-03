class AddAttachmentImageToPages < ActiveRecord::Migration[4.2]
  def self.up
    change_table :pages do |t|
      t.attachment :image
    end
  end

  def self.down
    remove_attachment :pages, :image
  end
end

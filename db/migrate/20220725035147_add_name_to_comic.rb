class AddNameToComic < ActiveRecord::Migration[7.0]
  def change
    add_column :comics, :name, :string
  end
end

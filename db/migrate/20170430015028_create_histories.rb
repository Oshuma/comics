class CreateHistories < ActiveRecord::Migration[5.0]
  def change
    create_table :histories do |t|
      t.references :user, foreign_key: true
      t.string :group_name
      t.string :comic_name

      t.timestamps
    end
  end
end

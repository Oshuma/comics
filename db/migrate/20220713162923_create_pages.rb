class CreatePages < ActiveRecord::Migration[7.0]
  def change
    create_table :pages do |t|
      t.integer :number
      t.boolean :read, default: false
      t.references :comic, null: false, foreign_key: true

      t.timestamps
    end
  end
end

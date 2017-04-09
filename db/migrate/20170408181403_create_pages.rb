class CreatePages < ActiveRecord::Migration[5.0]
  def change
    create_table :pages do |t|
      t.integer :number
      t.boolean :read, default: false
      t.references :comic, foreign_key: true

      t.timestamps
    end
  end
end

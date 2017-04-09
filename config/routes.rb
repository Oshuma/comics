Rails.application.routes.draw do
  devise_for :users

  resources :groups do
    delete :delete_comics, on: :member
  end

  resources :comics do
    collection do
      delete :delete_read
    end

    member do
      get :read
      put :finish
      put :move
    end

    resources :pages do
      member do
        put :next_page
        put :previous_page
      end
    end
  end

  get 'stats' => 'stats#index'

  root 'groups#index'
end

Rails.application.routes.draw do
  # Define your application routes per the DSL in https://guides.rubyonrails.org/routing.html

  get 'login' => 'user_sessions#new', :as => :login
  post 'login' => "user_sessions#create"
  delete 'logout' => 'user_sessions#destroy', :as => :logout

  resources :groups do
    delete :delete_comics, on: :member
    delete :delete_read, on: :member
  end

  resources :comics do
    collection do
      delete :delete_read
    end

    member do
      get :read
      put :finish
    end

    resources :pages do
      member do
        put :next_page
        put :previous_page
      end
    end
  end

  get 'history' => 'history#index'
  get 'stats' => 'stats#index'

  get 'setup' => 'setup#new'
  post 'setup' => 'setup#create'

  get 'admin' => 'admin#index'
  namespace :admin do
    resources :users do
      member do
        patch :enable_admin
        patch :disable_admin
      end
    end
  end

  root 'groups#index'
end

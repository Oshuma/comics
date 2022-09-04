def ensure_image_exists!(tag)
  if `docker image inspect #{tag} 2>/dev/null`.chomp == "[]"
    puts "\n==Cache image '#{tag}' not found. Building..."
    system("docker build --tag #{tag} .", exception: true)
  end
end

namespace :docker do
  image_latest = "comics:latest"
  image_cache = "comics:gem-cache"

  task image: 'image:build'.to_sym

  namespace :image do
    desc "Build #{image_latest} image"
    task :build do
      ensure_image_exists!(image_cache)
      system("docker build --tag #{image_latest} --build-arg CACHE_IMAGE=#{image_cache} .", exception: true)
    end

    desc "Build #{image_cache} gem cache image"
    task :cache do
      ensure_image_exists!(image_latest)
      system("docker image tag #{image_latest} #{image_cache}", exception: true)
    end

    desc "Scan #{image_latest} and #{image_cache} for vulnerabilities"
    task :scan do
      [image_latest, image_cache].each do |i|
        system("docker scan --accept-license #{i}", exception: true)
      end
    end

    desc "Remove #{image_latest} image"
    task :clean do
      system("docker rmi #{image_latest}", exception: true)
    end

    desc "Remove #{image_cache} gem cache image"
    task "clean:cache".to_sym do
      system("docker rmi #{image_cache}", exception: true)
    end
  end
end

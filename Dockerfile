ARG BASE_IMAGE=ruby:2.7-alpine
ARG CACHE_IMAGE=${BASE_IMAGE}

# Create a build stage for the gem cache
# Ensure that the /usr/local/bundle exists in case we use an empty image as cache
FROM ${CACHE_IMAGE} AS gem-cache
RUN mkdir -p /usr/local/bundle

# Create an intermediate image that has bundler installed
FROM $BASE_IMAGE AS base
RUN gem install bundler:2.2.9
WORKDIR /app

# Copy the gems from a the gem-cache build stage, install missing gems and clean up
FROM base AS gems
RUN apk add --update --no-cache \
      build-base \
      file \
      imagemagick \
      postgresql-dev \
      tzdata \
      vips \
      && rm -rf /var/cache/apk/*

COPY --from=gem-cache /usr/local/bundle /usr/local/bundle
COPY Gemfile Gemfile.lock ./
RUN bundle install && bundle clean --force

# Copy the gems from the gems build stage and get the source code in place
FROM base AS final
COPY --from=gems /usr/local/bundle /usr/local/bundle

ARG GIT_VERSION
ARG BUILD_DATE

LABEL io.github.oshuma.comics.git-version=$GIT_VERSION \
      io.github.oshuma.comics.build-date=$BUILD_DATE

LABEL org.label-schema.vcs-ref=$GIT_VERSION \
      org.label-schema.vcs-url="https://github.com/Oshuma/comics"

RUN apk add --update --no-cache \
      build-base \
      file \
      imagemagick \
      nodejs \
      postgresql-dev \
      tzdata \
      unzip \
      vips \
      yarn \
      && rm -rf /var/cache/apk/*

ADD https://www.rarlab.com/rar/unrarsrc-6.1.7.tar.gz .
RUN tar xzf unrarsrc-6.1.7.tar.gz \
      && cd unrar/ \
      && make && make install \
      && rm -rf unrar/ unrarsrc-6.1.7.tar.gz

# make 'docker logs' work
ENV RAILS_LOG_TO_STDOUT=true

WORKDIR /app
COPY . .

ENV RAILS_SERVE_STATIC_FILES=true
RUN yarn install \
      && yarn build \
      && yarn build:css \
      && bin/rails assets:precompile

EXPOSE 3000/tcp

CMD ["/app/docker-entrypoint.sh"]

FROM ruby:2.3-alpine

RUN apk add --update --no-cache \
      build-base \
      file \
      imagemagick \
      nodejs \
      postgresql-dev \
      tzdata \
      unrar \
      unzip \
    && rm -rf /var/cache/apk/*

RUN mkdir /app
WORKDIR /app

ADD Gemfile /app/Gemfile
ADD Gemfile.lock /app/Gemfile.lock

RUN bundle install

ADD . /app

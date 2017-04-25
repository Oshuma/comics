FROM ruby:2.3-alpine

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
      unrar \
      unzip \
    && rm -rf /var/cache/apk/*

RUN mkdir /app
WORKDIR /app

ADD Gemfile /app/Gemfile
ADD Gemfile.lock /app/Gemfile.lock

RUN bundle install

ADD . /app

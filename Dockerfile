FROM ruby:2-alpine

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
      wget \
    && rm -rf /var/cache/apk/*

# Grab and compile unrar source from here: https://www.rarlab.com/rar_add.htm
RUN wget -O - https://www.rarlab.com/rar/unrarsrc-6.1.7.tar.gz | tar xz \
      && cd unrar/ \
      && make && make install \
      && rm -rf unrar/

RUN mkdir /app
WORKDIR /app

ADD Gemfile /app/Gemfile
ADD Gemfile.lock /app/Gemfile.lock

RUN bundle install

ADD . /app

EXPOSE 3000/tcp

CMD ["/app/docker-entrypoint.sh"]

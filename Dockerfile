FROM ruby:2.4.1

# Enable non-free repo, so we get an 'unrar' that acts like we expect.
RUN sed -i "s/jessie main/jessie main contrib non-free/" /etc/apt/sources.list

RUN apt-get update -qq \
    && apt-get install -y --no-install-recommends \
         build-essential \
         imagemagick \
         libpq-dev \
         nodejs \
         unrar \
         unzip \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir /app
WORKDIR /app

ADD Gemfile /app/Gemfile
ADD Gemfile.lock /app/Gemfile.lock

RUN bundle install

ADD . /app

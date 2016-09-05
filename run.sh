#!/bin/sh

# This is only used when running locally.

# We need to run sudo since mongod needs it to run.
sudo echo 'Starting...'
foreman start --procfile Procfile.dev

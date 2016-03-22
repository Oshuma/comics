#!/bin/sh

# We need to run sudo since mongod needs it to run.
sudo echo 'Starting...'
foreman start

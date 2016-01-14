#!/bin/sh

docker build -t "knollit/"$CIRCLE_PROJECT_REPONAME"_rdbms:latest" .

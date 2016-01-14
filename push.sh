#!/bin/sh

docker tag knollit/$CIRCLE_PROJECT_REPONAME:latest knollit/$CIRCLE_PROJECT_REPONAME:$CIRCLE_SHA1
docker push knollit/$CIRCLE_PROJECT_REPONAME:$CIRCLE_SHA1
docker push knollit/$CIRCLE_PROJECT_REPONAME:latest

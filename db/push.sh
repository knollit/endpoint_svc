#!/bin/sh

repo="knollit/"$CIRCLE_PROJECT_REPONAME"_rdbms:"
docker tag $repo"latest" $repo$CIRCLE_SHA1
docker push $repo$CIRCLE_SHA1
docker push $repo"latest"

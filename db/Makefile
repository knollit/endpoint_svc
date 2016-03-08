repo = knollit/endpoint_svc_rdbms

all: build

build:
	docker build -t $(repo):latest .

publish: build
	docker tag $(repo):latest $(repo):$$CIRCLE_SHA1
	docker push $(repo):$$CIRCLE_SHA1
	docker push $(repo):latest

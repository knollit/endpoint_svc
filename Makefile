all: build

build: flatbuffers
	CGO_ENABLED=0 GOOS=linux go build -a --installsuffix cgo --ldflags="-s" -o dest/endpoint_svc .
	docker build -t knollit/endpoint_svc:latest .

flatbuffers:
	flatc -g -o $${GOPATH##*:}/src/github.com/knollit/endpoint_svc *.fbs

clean:
	rm -rf dest

publish: build
	docker tag knollit/endpoint_svc:latest knollit/endpoint_svc:$$CIRCLE_SHA1
	docker push knollit/endpoint_svc:$$CIRCLE_SHA1
	docker push knollit/endpoint_svc:latest

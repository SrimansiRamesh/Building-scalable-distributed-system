# Build the Lambda binary and zip it for deployment.
# Must be linux/amd64 regardless of your local machine (Apple Silicon or otherwise).

.PHONY: build clean

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap .
	zip function.zip bootstrap
	rm bootstrap

clean:
	rm -f bootstrap function.zip

BINARY_NAME=bin/gopherspy
MAIN_PATH=cmd/main.go

## help: Show help menu
help:
	@echo "Available commands:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

usage:
	@go run ${MAIN_PATH} -h

## build: Compile the project into a binary executable
build:
	@echo "Compiling the binary..."
	go build -o ${BINARY_NAME} ${MAIN_PATH}

## example: Run the project with example URLs and output to console
example:
	go run ${MAIN_PATH} https://google.com https://go.dev

## example-txt: Run the project saving the results to a txt file
example-txt:
	go run ${MAIN_PATH} -format=txt -rate=5 -input=examples.txt

## example-json: Run the project saving the results to a json file
example-json:
	go run ${MAIN_PATH} -format=json -rate=5 -input=examples.txt

## example-xml: Run the project saving the results to an xml file
example-xml:
	go run ${MAIN_PATH} -format=xml -rate=5 -input=examples.txt

## clean: Remove the binary and result files
clean:
	@echo "Cleaning temporary files..."
	rm -f ${BINARY_NAME}
	rm -f results.txt
	rm -f results.json
	rm -f results.xml

## test: Run unit tests
test:
	go test ./... -v
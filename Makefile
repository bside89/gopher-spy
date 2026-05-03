BINARY_NAME=bin/gopherspy
MAIN_PATH=main.go

## help: Show help menu
help:
	@echo "Available commands:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## build: Compile the project into a binary executable
build:
	@echo "Compiling the binary..."
	go build -o ${BINARY_NAME} ${MAIN_PATH}

## example: Run the project with example URLs and output to console
example:
	go run ${MAIN_PATH} https://google.com https://go.dev

## example-file: Run the project saving the results to a file
example-file:
	go run ${MAIN_PATH} -file https://google.com https://github.com

## clean: Remove the binary and result files
clean:
	@echo "Cleaning temporary files..."
	rm -f ${BINARY_NAME}
	rm -f results.txt

## test: Run unit tests
test:
	go test ./... -v
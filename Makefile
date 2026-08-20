BINARY_NAME=threader

build:
	bash install.sh

test:
	go test -v ./...

clean:
	rm -f $(BINARY_NAME)

run: build
	./$(BINARY_NAME)

deps-linux:
	sudo apt-get update && sudo apt-get install -y libtesseract-dev libleptonica-dev tesseract-ocr

setup:
	go mod download

BINARY_NAME=threader

build:
	go build -o $(BINARY_NAME) ./cmd/threader

clean:
	rm -f $(BINARY_NAME)

run: build
	./$(BINARY_NAME)

deps-linux:
	sudo apt-get update && sudo apt-get install -y libtesseract-dev libleptonica-dev tesseract-ocr

setup:
	go mod download

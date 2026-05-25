.PHONY: test build run clean

test:
	go test ./core/... ./cli/...

build:
	go build -o bin/finops ./cli/cmd/finops

run: build
	./bin/finops hello

clean:
	rm -rf bin dist

# Ad-hoc cross-compile examples:
# GOOS=linux GOARCH=amd64 go build -o bin/finops-linux-amd64 ./cli/cmd/finops
# GOOS=windows GOARCH=amd64 go build -o bin/finops.exe ./cli/cmd/finops

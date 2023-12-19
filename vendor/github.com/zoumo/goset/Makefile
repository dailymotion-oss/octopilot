
test:
	go list ./... | grep -v '/vendor/' | grep -v '/tests/' | xargs go test 

bench:
	go list ./... | grep -v '/vendor/' | grep -v '/tests/' | xargs go test -bench=.

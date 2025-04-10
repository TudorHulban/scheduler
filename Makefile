today:
	@grep "cpu MHz" /proc/cpuinfo | awk -F: '{print $$2}' | awk '{printf "Core %2d: %7.2f MHz\n", NR-1, $$1}'

test-full:
	@CGO_ENABLED=1 go test -count=1 ./... -json -cover -race > test-output.json; \
	tparse -smallscreen -file test-output.json; \
	jq -r 'select(.Action == "fail" or .Action == "pass") | select(.Test != null) | .Action' test-output.json | sort | uniq -c | awk '{print $$2 ": " $$1}' | xargs echo "Summary:"; \
	rm -f test-output.json
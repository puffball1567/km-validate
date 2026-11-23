#!/bin/sh
set -eu

project_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
test_dir=$(mktemp -d)
trap 'rm -rf "$test_dir"' EXIT HUP INT TERM

keika_command=${KEIKA:-keika}
go_cache="$test_dir/go-cache"
generated_dir="$test_dir/generated"
mkdir -p "$go_cache" "$generated_dir"

cd "$project_dir"
"$keika_command" deps lock
GOCACHE="${GOCACHE:-$test_dir/go-cache}" go test "$project_dir/scripts/check_coverage.go" "$project_dir/scripts/check_coverage_test.go"
"$keika_command" check validate.km
"$keika_command" check examples/basic/main.km
"$keika_command" check examples/forms/main.km
"$keika_command" check tests/fixtures/database_lookup.km
"$keika_command" emit-go -package validate -o "$generated_dir/validate.go" -source-map "$generated_dir/validate.map.json" validate.km
cp tests/*_test.go "$generated_dir/"

cd "$generated_dir"
go mod init example.com/km-validate-test >/dev/null
GOCACHE="$go_cache" go vet ./...
GOCACHE="$go_cache" go test -count=1 -race -coverprofile=coverage.out ./...
GOCACHE="$go_cache" go run "$project_dir/scripts/check_coverage.go" -- "$generated_dir/validate.go" "$generated_dir/validate.map.json" "$generated_dir/coverage.out"
GOCACHE="$go_cache" go test -run='^$' -fuzz='^FuzzRejectIsAtomic$' -fuzztime=1s
GOCACHE="$go_cache" go test -run='^$' -fuzz='^FuzzTextRulesNeverPanic$' -fuzztime=1s
GOCACHE="$go_cache" go test -run='^$' -fuzz='^FuzzStructuredRulesNeverPanic$' -fuzztime=1s
GOCACHE="$go_cache" go test -run='^$' -fuzz='^FuzzAddAllIsAtomic$' -fuzztime=1s
GOCACHE="$go_cache" go test -run='^$' -fuzz='^FuzzComparisonRulesNeverPanic$' -fuzztime=1s

GOCACHE="$go_cache" go test -run='^$' -fuzz='^FuzzEachIsAtomic$' -fuzztime=1s
GOCACHE="$go_cache" go test -run='^$' -fuzz='^FuzzFileRuleIsAtomic$' -fuzztime=1s

cd "$project_dir"
"$keika_command" build -o "$test_dir/basic-example" examples/basic/main.km
actual_output=$("$test_dir/basic-example")
expected_output='user.name:required
user.email:email
user.age:integer_range'
if [ "$actual_output" != "$expected_output" ]; then
  printf 'unexpected basic example output:\n%s\n' "$actual_output" >&2
  exit 1
fi

"$keika_command" build -o "$test_dir/forms-example" examples/forms/main.km
actual_output=$("$test_dir/forms-example")
expected_output='company:required_if
password_confirmation:same
end:compare_integer
contacts.1:email
avatar:file_type
email:unique
team_id:exists'
if [ "$actual_output" != "$expected_output" ]; then
  printf 'unexpected forms example output:\n%s\n' "$actual_output" >&2
  exit 1
fi

library_dir="$test_dir/library"
consumer_dir="$test_dir/external-consumer"
mkdir "$library_dir"
cp validate.km go.mod kinmokusei.lock kinmokusei.toml LICENSE "$library_dir/"
"$keika_command" new app --module example.com/km-validate-consumer "$consumer_dir"
cp tests/external_consumer/main.km "$consumer_dir/main.km"
cd "$consumer_dir"
"$keika_command" deps add --offline --replace ../library github.com/puffball1567/km-validate@v0.3.1
"$keika_command" check
"$keika_command" build -o "$test_dir/external-consumer-app"
actual_output=$("$test_dir/external-consumer-app")
if [ "$actual_output" != "valid" ]; then
  printf 'unexpected external-consumer output:\n%s\n' "$actual_output" >&2
  exit 1
fi

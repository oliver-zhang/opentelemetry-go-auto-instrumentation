# Use extensions to detect sql injection

The document demonstrates how to use extensions to detect sql injection.

## Step1: Replace path in config.json
Replace the `Path` in `config.json` with the actual absolute path of `rules` directory, for example:
``` json
[{
"ImportPath": "database/sql",
"Function": "Query",
"ReceiverType": "*DB",
"OnEnter": "sqlQueryOnEnter",
"Path": "/Users/liuziming/Desktop/opentelemetry-go-auto-instrumentation/example/extension/sqlinject/rules"
}]
```

## Step2: Compile the target binary with otel
Use `otel` to build the binary with `config.json`:
```
cd example/extension/sqlinject
../../../otel set -rule=config.json
../../../otel go build demo/sql_inject.go
```
Users can get the `otel` according to [documentation](../../../README.md)

## Step3: Run the binary compiled by otel
```shell
docker run -d -p 3306:3306 -p 33060:33060 -e MYSQL_USER=test -e MYSQL_PASSWORD=test -e MYSQL_DATABASE=test -e MYSQL_ALLOW_EMPTY_PASSWORD=yes mysql:8.0.36
./sql_inject
```
And the result will be:
```shell
2024/11/04 21:24:55 sqlQueryOnEnter potential SQL injection detected
```
It means that the sql injection is detected correctly.

package main

//go:generate mkdir -p ./avro
//go:generate $GOPATH/bin/gogen-avro ./avro version.avsc

@echo off
go test -coverprofile=coverage.txt -v ./...
go tool cover -html=coverage.txt
del coverage.txt

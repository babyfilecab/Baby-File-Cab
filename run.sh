#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"
go mod tidy
wails dev

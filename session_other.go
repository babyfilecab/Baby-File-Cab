//go:build !linux

package main

import "context"

func (a *App) startSessionMonitor(ctx context.Context) {}

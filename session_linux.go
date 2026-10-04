//go:build linux

package main

import (
	"context"
	"os"
	"github.com/godbus/dbus/v5"
)

func (a *App) startSessionMonitor(ctx context.Context) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return
	}
	var sessionPath dbus.ObjectPath
	// Match only this process's desktop session, not other users' sessions.
	call := conn.Object("org.freedesktop.login1", dbus.ObjectPath("/org/freedesktop/login1")).Call("org.freedesktop.login1.Manager.GetSessionByPID", 0, uint32(os.Getpid()))
	_ = call.Store(&sessionPath)
	_ = conn.AddMatchSignal(dbus.WithMatchInterface("org.freedesktop.login1.Manager"), dbus.WithMatchMember("PrepareForSleep"))
	_ = conn.AddMatchSignal(dbus.WithMatchInterface("org.freedesktop.login1.Session"), dbus.WithMatchMember("Lock"))
	_ = conn.AddMatchSignal(dbus.WithMatchInterface("org.freedesktop.DBus.Properties"), dbus.WithMatchMember("PropertiesChanged"), dbus.WithMatchArg(0, "org.freedesktop.login1.Session"))
	signals := make(chan *dbus.Signal, 16)
	conn.Signal(signals)
	go func() {
		defer conn.RemoveSignal(signals)
		for {
			select {
			case <-ctx.Done():
				return
			case signal := <-signals:
				if signal == nil {
					continue
				}
				switch signal.Name {
				case "org.freedesktop.login1.Manager.PrepareForSleep":
					if len(signal.Body) > 0 {
						if sleeping, ok := signal.Body[0].(bool); ok && sleeping {
							a.Logout()
						}
					}
				case "org.freedesktop.login1.Session.Lock":
					if sessionPath != "" && signal.Path == sessionPath {
						a.Logout()
					}
				case "org.freedesktop.DBus.Properties.PropertiesChanged":
					if sessionPath == "" || signal.Path != sessionPath || len(signal.Body) < 2 {
						continue
					}
					props, ok := signal.Body[1].(map[string]dbus.Variant)
					if !ok {
						continue
					}
					if active, ok := props["Active"]; ok {
						if on, ok := active.Value().(bool); ok && !on {
							a.Logout()
						}
					}
				}
			}
		}
	}()
}

package main

import (
	"encoding/json"
	"flag"
	"github.com/asticode/go-astilectron"
)

var (
	w              *astilectron.Window    // 当前程序的窗口
	intercomW      *astilectron.Window
	debug            = flag.Bool("d", false, "enables the debug mode")
)

// <<<<<<<<<<<<<<<<< bootstrap 相关的数据结构 >>>>>>>>>>>>>>>>>>>

// AstilectronAdapter is a function that adapts the astilectron instance
type AstilectronAdapter func(w *astilectron.Astilectron)

// Asset is a function that retrieves an asset content namely the go-bindata's Asset method
type AssetFunc func(name string) ([]byte, error)

// AssetDir is a function that retrieves an asset dir namely the go-bindata's AssetDir method
type AssetDirFunc func(name string) ([]string, error)

// MessageHandler is a functions that handles messages
type MessageHandler func(w *astilectron.Window, m MessageIn) (payload interface{}, err error)

// OnWait is a function that executes custom actions before waiting
type OnWait func(a *astilectron.Astilectron, w []*astilectron.Window, m *astilectron.Menu, t *astilectron.Tray, tm *astilectron.Menu) error

// RestoreAssets is a function that restores assets namely the go-bindata's RestoreAssets method
type RestoreAssetsFunc func(dir, name string) error

// WindowAdapter is a function that adapts a window
type WindowAdapter func(w *astilectron.Window)

// Options represents options
type Options struct {
	Adapter            AstilectronAdapter
	Asset              AssetFunc
	AssetDir           AssetDirFunc
	AstilectronOptions astilectron.Options
	Debug              bool
	MenuOptions        []*astilectron.MenuItemOptions
	OnWait             OnWait
	ResourcesPath      string
	RestoreAssets      RestoreAssetsFunc
	TrayMenuOptions    []*astilectron.MenuItemOptions
	TrayOptions        *astilectron.TrayOptions
	Windows            []*Window
}

// Options to setup and create a new window
type Window struct {
	Adapter        WindowAdapter
	Homepage       string
	MessageHandler MessageHandler
	Options        *astilectron.WindowOptions
}

// MessageOut represents a message going out
type MessageOut struct {
	Name    string      `json:"name"`
	Payload interface{} `json:"payload,omitempty"`
}

// MessageIn represents a message going in
type MessageIn struct {
	Name    string          `json:"name"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

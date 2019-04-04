package main

import (
	"bifrost/log"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilectron-bundler"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/ptr"
	"github.com/pkg/errors"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

var listener net.Listener
var a *astilectron.Astilectron

// HandleMessages handles messages
func HandleMessages(w *astilectron.Window, messageHandler MessageHandler) astilectron.ListenerMessage {
	return func(m *astilectron.EventMessage) (v interface{}) {
		// Unmarshal message
		var i MessageIn
		var err error
		if err = m.Unmarshal(&i); err != nil {
			astilog.Error(errors.Wrapf(err, "unmarshaling message %+v failed", *m))
			return
		}

		// Handle message
		var p interface{}
		if p, err = messageHandler(w, i); err != nil {
			astilog.Error(errors.Wrapf(err, "handling message %+v failed", i))
		}

		// Return message
		if p != nil {
			o := &MessageOut{Name: i.Name + ".callback", Payload: p}
			if err != nil {
				o.Name = "error"
			}
			v = o
		}
		return
	}
}

// CallbackMessage represents a bootstrap message callback
type CallbackMessage func(m *MessageIn)

// SendMessage sends a message
func SendMessage(w *astilectron.Window, name string, payload interface{}, cs ...CallbackMessage) error {
	var callbacks []astilectron.CallbackMessage
	for _, c := range cs {
		callbacks = append(callbacks, func(e *astilectron.EventMessage) {
			var m *MessageIn
			if e != nil {
				m = &MessageIn{}
				if err := e.Unmarshal(m); err != nil {
					astilog.Error(errors.Wrap(err, "unmarshaling event message failed"))
					return
				}
			}
			c(m)
		})
	}
	return w.SendMessage(MessageOut{Name: name, Payload: payload}, callbacks...)
}

// Run runs the bootstrap
func Run(o Options) (err error) {
	// Create astilectron
	log.Info("--------------astilectron.New()----------------------------------------")

	if a, err = astilectron.New(o.AstilectronOptions); err != nil {
		return errors.Wrap(err, "creating new astilectron failed")
	}
	defer a.Close()

	// Handle signals
	a.HandleSignals()

	// Adapt astilectron
	if o.Adapter != nil {
		o.Adapter(a)
	}

	// Set provisioner
	if o.Asset != nil {
		a.SetProvisioner(astibundler.NewProvisioner(o.Asset))
	}

	// Get relative and absolute resources path
	var relativeResourcesPath = o.ResourcesPath
	if len(relativeResourcesPath) == 0 {
		relativeResourcesPath = "resources"
	}

	// Restore resources
	if o.RestoreAssets != nil {
		if err = RestoreResources(a, o.Asset, o.AssetDir, o.RestoreAssets, relativeResourcesPath); err != nil {
			err = errors.Wrap(err, "restoring resources failed")
			return
		}
	}
	log.Info("--------------astilectron.Strat()-----------------------------------------")
	// Start
	if err = a.Start(); err != nil {
		return errors.Wrap(err, "starting astilectron failed")
	}

	// Init windows
	var w = make([]*astilectron.Window, len(o.Windows))
	log.Info("run():", "start window.........")
	for i, wo := range o.Windows {
		if listener == nil {
			buff := make([]uint16, syscall.MAX_PATH)
			syscall.MustLoadDLL("Shell32.dll").MustFindProc("SHGetFolderPathW").Call(0, 26, 0, 0, uintptr(unsafe.Pointer(&buff[0])))
			if listener, err = net.Listen("tcp", "127.0.0.1:"); err != nil {
				return errors.Wrap(err, "net.Listen failed")
			}
			go http.Serve(listener, http.FileServer(http.Dir(syscall.UTF16ToString(buff)+`\Test\resources\dist\app\`)))
			log.Info("run():", "listen() 0k")
		}
		time.Sleep(500 * time.Millisecond)
		if w[i], err = a.NewWindow("http://" + path.Join(listener.Addr().String(), wo.Homepage), wo.Options); err != nil {
			return errors.Wrap(err, "new window failed")
		}
		log.Info("run():", "newWindow() 0k")

		// Handle messages
		if wo.MessageHandler != nil {
			w[i].OnMessage(HandleMessages(w[i], wo.MessageHandler))
		}

		// Adapt window
		if wo.Adapter != nil {
			wo.Adapter(w[i])
		}

		// Create window
		if err = w[i].Create(); err != nil {
			return errors.Wrap(err, "creating window failed")
		}
		log.Info("run():", "create() 0k")
		w[i].Show()
	}

	// Debug
	if o.Debug {
		// Create menu item
		var debug bool
		mi := &astilectron.MenuItemOptions{
			Accelerator: astilectron.NewAccelerator("Control", "d"),
			Label:       astiptr.Str("Debug"),
			OnClick: func(e astilectron.Event) (deleteListener bool) {
				for i, window := range w {
					width := *o.Windows[i].Options.Width
					if debug {
						if err := window.CloseDevTools(); err != nil {
							astilog.Error(errors.Wrap(err, "closing dev tools failed"))
						}
						if err := window.Resize(width, *o.Windows[i].Options.Height); err != nil {
							astilog.Error(errors.Wrap(err, "resizing window failed"))
						}
					} else {
						if err := window.OpenDevTools(); err != nil {
							astilog.Error(errors.Wrap(err, "opening dev tools failed"))
						}
						if err := window.Resize(width+700, *o.Windows[i].Options.Height); err != nil {
							astilog.Error(errors.Wrap(err, "resizing window failed"))
						}
					}
				}
				debug = !debug
				return
			},
			Type: astilectron.MenuItemTypeCheckbox,
		}

		// Add menu item
		if len(o.MenuOptions) == 0 {
			o.MenuOptions = []*astilectron.MenuItemOptions{{SubMenu: []*astilectron.MenuItemOptions{mi}}}
		} else {
			if len(o.MenuOptions[0].SubMenu) > 0 {
				o.MenuOptions[0].SubMenu = append(o.MenuOptions[0].SubMenu, &astilectron.MenuItemOptions{Type: astilectron.MenuItemTypeSeparator})
			}
			o.MenuOptions[0].SubMenu = append(o.MenuOptions[0].SubMenu, mi)
		}
	}

	// Menu
	var m *astilectron.Menu
	if *debug && (len(o.MenuOptions) > 0) {
		// Init menu
		m = a.NewMenu(o.MenuOptions)

		// Create menu
		if err = m.Create(); err != nil {
			return errors.Wrap(err, "creating menu failed")
		}
	}

	// Tray
	var t *astilectron.Tray
	var tm *astilectron.Menu
	if o.TrayOptions != nil {
		// Make sure path to image is absolute
		if o.TrayOptions.Image != nil && !filepath.IsAbs(*o.TrayOptions.Image) {
			*o.TrayOptions.Image = filepath.Join(a.Paths().DataDirectory(), *o.TrayOptions.Image)
		}

		// Init tray
		t = a.NewTray(o.TrayOptions)

		// Create tray
		if err = t.Create(); err != nil {
			return errors.Wrap(err, "creating tray failed")
		}

		// Tray menu
		if len(o.TrayMenuOptions) > 0 {
			// Init tray menu
			tm = t.NewMenu(o.TrayMenuOptions)

			// Create tray menu
			if err = tm.Create(); err != nil {
				return errors.Wrap(err, "creating tray menu failed")
			}
		}
	}

	// On wait
	if o.OnWait != nil {
		if err = o.OnWait(a, w, m, t, tm); err != nil {
			return errors.Wrap(err, "onwait failed")
		}
	}

	// Blocking pattern
	a.Wait()
	return
}

func absoluteResourcesPath(a *astilectron.Astilectron, relativeResourcesPath string) string {
	return filepath.Join(a.Paths().DataDirectory(), relativeResourcesPath)
}

func RestoreResources(a *astilectron.Astilectron, asset AssetFunc, assetDir AssetDirFunc, assetRestorer RestoreAssetsFunc, relativeResourcesPath string) (err error) {
	// Check resources
	var restore bool
	var computedChecksums map[string]string
	var checksumsPath string
	if restore, computedChecksums, checksumsPath, err = checkResources(a, asset, assetDir, relativeResourcesPath); err != nil {
		err = errors.Wrap(err, "checking resources failed")
		return
	}

	// Restore resources
	if restore {
		if err = restoreResources(a, relativeResourcesPath, assetRestorer, computedChecksums, checksumsPath); err != nil {
			err = errors.Wrap(err, "restoring resources failed")
			return
		}
	} else {
		astilog.Debug("Skipping restoring resources...")
	}
	return
}

func checkResources(a *astilectron.Astilectron, asset AssetFunc, assetDir AssetDirFunc, relativeResourcesPath string) (restore bool, computedChecksums map[string]string, checksumsPath string, err error) {
	// Compute checksums
	arp := absoluteResourcesPath(a, relativeResourcesPath)
	checksumsPath = filepath.Join(arp, "checksums.json")
	if asset != nil && assetDir != nil {
		computedChecksums = make(map[string]string)
		if err = checksumAssets(asset, assetDir, relativeResourcesPath, computedChecksums); err != nil {
			err = errors.Wrap(err, "getting checksum of assets failed")
			return
		}
	}

	// Stat resources
	if _, err = os.Stat(arp); err != nil && !os.IsNotExist(err) {
		err = errors.Wrapf(err, "stating %s failed", arp)
		return
	} else if os.IsNotExist(err) {
		astilog.Debug("Resources folder doesn't exist, restoring resources...")
		err = nil
		restore = true
		return
	}

	// No computed checksums
	if computedChecksums == nil {
		astilog.Debug("No computed checksums, restoring resources...")
		restore = true
		return
	}

	// Stat checksums file
	if _, err = os.Stat(checksumsPath); err != nil && !os.IsNotExist(err) {
		err = errors.Wrapf(err, "stating %s failed", checksumsPath)
		return
	} else if os.IsNotExist(err) {
		astilog.Debug("Checksums file doesn't exist, restoring resources...")
		err = nil
		restore = true
		return
	}

	// Open resources checksums
	var f *os.File
	if f, err = os.Open(checksumsPath); err != nil {
		err = errors.Wrapf(err, "opening %s failed")
		return
	}
	defer f.Close()

	// Unmarshal checksums
	var unmarshaledChecksums map[string]string
	if err = json.NewDecoder(f).Decode(&unmarshaledChecksums); err != nil {
		err = errors.Wrap(err, "unmarshaling checksums failed")
		return
	}

	// Check number of paths
	if len(unmarshaledChecksums) != len(computedChecksums) {
		astilog.Debugf("%d paths in unmarshaled checksums != %d paths in computed checksums, restoring resources...", len(unmarshaledChecksums), len(computedChecksums))
		restore = true
		return
	}

	// Loop through computed checksums
	for p, c := range computedChecksums {
		// Path doesn't exist in unmarshaled checksums
		v, ok := unmarshaledChecksums[p]
		if !ok {
			astilog.Debugf("Path %s doesn't exist in unmarshaled checksums, restoring resources...", p)
			restore = true
			return
		}

		// Checksums are different
		if c != v {
			astilog.Debugf("Unmarshaled checksum (%s) != computed checksum (%s) for path %s, restoring resources...", v, c, p)
			restore = true
			return
		}
	}
	return
}

func checksumAssets(asset AssetFunc, assetDir AssetDirFunc, name string, m map[string]string) (err error) {
	// Get children
	children, errDir := assetDir(name)

	// File
	if errDir != nil {
		// Get checksum
		var h string
		if h, err = checksumAsset(asset, name); err != nil {
			err = errors.Wrapf(err, "getting checksum of %s failed", name)
			return
		}
		m[name] = h
		return
	}

	// Dir
	for _, child := range children {
		if err = checksumAssets(asset, assetDir, filepath.Join(name, child), m); err != nil {
			err = errors.Wrapf(err, "getting checksum of assets in %s failed", name)
			return
		}
	}
	return
}

func checksumAsset(asset AssetFunc, name string) (o string, err error) {
	// Get data
	var b []byte
	if b, err = asset(name); err != nil {
		err = errors.Wrapf(err, "getting data from asset %s failed", name)
		return
	}

	// Hash
	h := md5.New()
	if _, err = h.Write(b); err != nil {
		err = errors.Wrapf(err, "writing data of asset %s to hash failed", name)
		return
	}
	o = base64.StdEncoding.EncodeToString(h.Sum(nil))
	return
}

func restoreResources(a *astilectron.Astilectron, relativeResourcesPath string, assetRestorer RestoreAssetsFunc, computedChecksums map[string]string, checksumsPath string) (err error) {
	// Remove resources
	arp := absoluteResourcesPath(a, relativeResourcesPath)
	astilog.Debugf("Removing %s", arp)
	if err = os.RemoveAll(arp); err != nil {
		err = errors.Wrapf(err, "removing %s failed", arp)
		return
	}

	// Restore resources
	astilog.Debugf("Restoring resources in %s", arp)
	if err = assetRestorer(a.Paths().DataDirectory(), relativeResourcesPath); err != nil {
		err = errors.Wrapf(err, "restoring resources in %s failed", arp)
		return
	}

	// Write checksums
	if computedChecksums != nil {
		// Create checksums file
		var f *os.File
		if f, err = os.Create(checksumsPath); err != nil {
			err = errors.Wrapf(err, "creating %s failed", checksumsPath)
			return
		}
		defer f.Close()

		// Marshal
		if err = json.NewEncoder(f).Encode(computedChecksums); err != nil {
			err = errors.Wrap(err, "marshaling checksums failed")
			return
		}
	}
	return
}

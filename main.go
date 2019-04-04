package main

import (
	"go-astilectron-demo/log"
	"flag"
	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilog"
	"github.com/marcsauter/single"
	"os"
)

// Vars
var (
	AppName       string

	singleInstant *single.Single
	prCookieKey   string         // panicwrap cookie key
	prCookieValue string         // panicwrap cookie value
)

type SoftwareNotifyData struct {
	Msg string `json:"msg"`
}

func main() {
	var windowsResizablePtr *bool
	//var windowsFramePtr     *bool
	var windowsHeightPtr    *int
	var windowsWidthPtr     *int

	flag.Parse()
	astilog.FlagInit()

	if *debug == true {
		windowsResizablePtr = astilectron.PtrBool(true)
		//windowsFramePtr = astilectron.PtrBool(true)
		windowsHeightPtr = astilectron.PtrInt(570 )
		windowsWidthPtr = astilectron.PtrInt(730 )
	} else {
		windowsResizablePtr = astilectron.PtrBool(false)
		//windowsFramePtr = astilectron.PtrBool(false)
		windowsHeightPtr = astilectron.PtrInt(500 )
		windowsWidthPtr = astilectron.PtrInt(700 )
	}

	// 启动窗口
	if err := Run(Options{
		Asset:    Asset,
		AssetDir: AssetDir,
		RestoreAssets: RestoreAssets,
		AstilectronOptions: astilectron.Options{
			AppName:            AppName,
			AppIconDarwinPath:  "resources/dist/icon.icns",
			AppIconDefaultPath: "resources/dist/icon.png",
			ElectronSwitches:   []string{"ignore-certificate-errors","true"},
		},
		Debug: *debug,
		ResourcesPath: "resources/dist",
		MenuOptions: []*astilectron.MenuItemOptions{{
			Label:   astilectron.PtrStr("Menu"),
		}},
		TrayOptions: &astilectron.TrayOptions {
			Image:   astilectron.PtrStr("resources/dist/icon.ico"),
			Tooltip: astilectron.PtrStr("Test Windows App"),
		},
		TrayMenuOptions: []*astilectron.MenuItemOptions {{
			Label: astilectron.PtrStr("quit"),
			OnClick: func(e astilectron.Event) (deleteListener bool) {
				ExitProgramWithoutPrompt()
				return
			},
		}},
		OnWait: func(_ *astilectron.Astilectron, ws []*astilectron.Window, _ *astilectron.Menu, t *astilectron.Tray, _ *astilectron.Menu) (err error) {
			w = ws[0]

			// 最小化到托盘
			w.On(astilectron.EventNameWindowEventMinimize, func(e astilectron.Event) (deleteListener bool) {
				_ = w.Hide()
				return
			})
			w.On(astilectron.EventNameWindowEventFocus, func(e astilectron.Event) (deleteListener bool) {
				_ = w.Show()
				return
			})

			if t != nil {
				// 注册托盘的双击切换显示/隐藏
				t.On(astilectron.EventNameTrayEventDoubleClicked, func(e astilectron.Event) (deleteListener bool) {
					if w.IsShown() {
						_ = w.Hide()
					} else {
						_ = w.Show()
					}
					return
				})
				// 注册托盘的单击聚焦功能
				t.On(astilectron.EventNameTrayEventClicked, func(e astilectron.Event) (deleteListener bool) {
					_ = w.Focus()
					return
				})
			}

			//-----------------------------------------------------------------------------------------------

			return nil
		},
		Windows: []*Window{{
			Homepage:       "index.html",
			MessageHandler: intercomHandleMessages,
			Options: &astilectron.WindowOptions{
				BackgroundColor: astilectron.PtrStr("#fff"),
				Center:          astilectron.PtrBool(false),
				Modal:           astilectron.PtrBool(true),
				Show:            astilectron.PtrBool(true),
				HasShadow:       astilectron.PtrBool(false),
				Resizable:       windowsResizablePtr,
				//Frame:           windowsFramePtr,
				Height:          windowsHeightPtr,
				Width:           windowsWidthPtr,
				//SkipTaskbar:     astilectron.PtrBool(true),
			},
		}},
	}); err != nil {
		log.Error("running bootstrap failed:", err)

	}

	// 收尾
	WrapUp()
}

// sendNotifyMessage 发送界面通知消息，处理严重错误
func sendNotifyMessage(name, msg string) {
	softwareNotifyData := SoftwareNotifyData{msg}
	if err := SendMessage(w, name, structToStr(softwareNotifyData)); err != nil {
		log.Error("sending event failed")
	}
}

// WrapUp 退出程序时做的一些收尾工作
func WrapUp() {
	// 关闭intercom窗口
	if intercomW != nil {
		if !intercomW.IsDestroyed() {
			intercomW.Destroy()
			//intercomA.Close()
			log.Info("intercom close ...................")
			intercomW = nil
		}
	}
	if w != nil {
		w.Destroy()
	}
}

// ExitProgramWithoutPrompt 程序退出，不会有退出的提示窗，但会执行WM_CLOSE，程序仍然正常退出
func ExitProgramWithoutPrompt() {
	if intercomW != nil {
		if !intercomW.IsDestroyed() {
			intercomW.Destroy()
			//intercomA.Close()
			log.Info("intercom close ...................")
			intercomW = nil
		}
	}
	if w != nil {
		_ = w.Destroy()
	}
	os.Exit(0)
}
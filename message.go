package main

import (
	"bifrost/log"
	"encoding/json"
	"fmt"
	"github.com/asticode/go-astilectron"
	"path"
	"strings"
)


/******************************* intercom window *************************************/
func intercomShow(intercomUrl string) (err error) {
	var oneW *astilectron.Window

	log.Info("-------------------------------------------------------")
	if intercomW == nil {

		log.Info("intercom url:", path.Join(listener.Addr().String(), intercomUrl))
		oneW, err = a.NewWindow("http://" + path.Join(listener.Addr().String(), intercomUrl), &astilectron.WindowOptions{
			BackgroundColor: astilectron.PtrStr("#fff"),
			Center:          astilectron.PtrBool(false),
			Modal:           astilectron.PtrBool(true),
			Show:            astilectron.PtrBool(false),
			HasShadow:       astilectron.PtrBool(false),
			Resizable:       astilectron.PtrBool(false),
			//Frame:           astilectron.PtrBool(false),
			Height:          astilectron.PtrInt(500),
			Width:           astilectron.PtrInt(560),
			//SkipTaskbar:     astilectron.PtrBool(true),
			Title: astilectron.PtrStr("intercom"),
		})
		/*
			windowsResizablePtr = astilectron.PtrBool(false)
			windowsFramePtr = astilectron.PtrBool(false)
			windowsHeightPtr = astilectron.PtrInt(500 )
			windowsWidthPtr = astilectron.PtrInt(700 )
		*/
		if err != nil {
			log.Error("newWindow():", err)
			return
		}

		// Handle messages
		oneW.OnMessage(HandleMessages(oneW, intercomHandleMessages))


		if err = oneW.Create(); err != nil {
			log.Error("newWindow():", err)
			return
		}
		oneW.On(astilectron.EventNameWindowEventClosed, func(e astilectron.Event) (deleteListener bool) {
			intercomW = nil
			return true
		})
		intercomW = oneW
	}

	if intercomW != nil {
		if !intercomW.IsDestroyed() {
			log.Info("intercom show ..................")
			intercomW.Show()
		}
	}

	return
}

// handleMessages handles messages
func intercomHandleMessages(win *astilectron.Window, m MessageIn) (payload interface{}, err error) {

	switch m.Name {
	case "intercomCmd":
		payload, err = intercomWndCmd(m)
	}

	return
}

func structToStr(st interface{}) string {
	retByteArr, _   := json.Marshal(&st)
	return string(retByteArr)
}
type CbBaseInfo struct {
	Status     string          `json:"status"`
	ErrMsg     string          `json:"errMsg"`
}

func msgToStr(message json.RawMessage) (jsonStr string) {

	jsonStr = string(message)
	// 去除字符串中间的 \
	jsonStr = strings.Replace(jsonStr, "\\", "", -1)
	// 去除字符串首尾的 "
	jsonStr = strings.Trim(jsonStr, "\"")
	return
}

func intercomWndCmd(m MessageIn) (payload string, err error) {

	var cbIntercom CbBaseInfo
	var scbbi = CbBaseInfo{"success", ""}

	type intercomCmd struct {
		 Cmd string `json:"cmd"`
		 Exension struct{
		 	Url string `json:"url"`
		 } `json:"extension"`
	}

	var wndcmd  intercomCmd

	jsonStr := msgToStr(m.Payload)
	//log.Info("intercomWndCmd():", jsonStr)
	if err = json.Unmarshal([]byte(jsonStr), &wndcmd); err != nil {
		log.Error(err)
		cbIntercom.Status = "failed"
		cbIntercom.ErrMsg = fmt.Sprint(err)
		payload = structToStr(cbIntercom)
		return
	}

	switch wndcmd.Cmd {
	case "show":
		err = intercomShow(wndcmd.Exension.Url)
		if err != nil {
			log.Error(err)
			cbIntercom.Status = "failed"
			cbIntercom.ErrMsg = fmt.Sprint(err)
			payload = structToStr(cbIntercom)
			return
		}
	case "close":
		if intercomW != nil {
			if !intercomW.IsDestroyed() {
				intercomW.Destroy()
				//intercomA.Close()
				log.Info("intercom close ...................")
				intercomW = nil
			}
		}
	default:
	}

	payload = structToStr(scbbi)
	return
}
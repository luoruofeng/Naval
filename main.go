package main

import (
	"github.com/luoruofeng/Naval/cmd"
	"github.com/luoruofeng/Naval/fx_opt"
)

func main() {
	cpmap := cmd.GetConfigFilePath()
	fxSrv := fx_opt.NewFxSrv(cpmap)
	fxSrv.Setup()
	fxSrv.Start()
	fxSrv.Shutddown()
}

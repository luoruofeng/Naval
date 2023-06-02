package main

import (
	"github.com/luoruofeng/Naval/cmd"
	"github.com/luoruofeng/Naval/fx_opt"
)

func main() {
	fxSrv := fx_opt.NewFxSrv(cmd.GetConfigFilePath())
	fxSrv.Setup()
	fxSrv.Start()
	fxSrv.Shutddown()
}

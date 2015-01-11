package main

import (
	"errors"
	"fmt"
	log "github.com/kdar/factorlog"
	"github.com/ksophocleous/rawman/rawmanlib"
	"os"
	"os/exec"
)

func convertDng(infile string, outfile string) error {
	cmd_dcraw := exec.Command("dcraw", "-c", fmt.Sprintf("%s", infile))
	cmd_convert := exec.Command("convert", "-", fmt.Sprintf("%s", outfile))

	cmd_convert.Stdin, _ = cmd_dcraw.StdoutPipe()

	err := cmd_convert.Start()
	if err != nil {
		return err
	}

	err = cmd_dcraw.Run()
	if err != nil {
		return errors.New(fmt.Sprintf("dcraw failed: %s", err.Error()))
	}

	err = cmd_convert.Wait()
	if err != nil {
		return errors.New(fmt.Sprintf("convert failed: %s", err.Error()))
	}

	return nil
}

func main() {
	rawman, err := rawmanlib.NewRawMan("./rawman.config")
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatal(err)
		} else {
			panic(err)
		}
	}
	defer rawman.Close()

	rawman.SetProcessFunc(convertDng)
	rawman.Loop()
}

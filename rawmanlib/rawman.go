package rawmanlib

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	log "github.com/kdar/factorlog"
	// "github.com/ksophocleous/rawman/rawmanlib"
	"golang.org/x/exp/fsnotify"
	//"log"
	"os"
	// "os/exec"
	"path"
	"path/filepath"
	"strings"
	// "sync"
	"time"
)

type rawMan struct {
	InputPath              string
	OutputPath             string
	OutputExt              string
	OutputFileMode         string
	OutputDirMode          string
	InputExt               string
	HysteresisUpdateTimeMs int

	// private fields
	watcher     *fsnotify.Watcher
	files       chan string
	shutdown    chan struct{}
	processFunc func(inputFilename string, outputFilename string) error
}

func (r *rawMan) SetProcessFunc(processFunc func(inputFilename string, outputFilename string) error) {
	r.processFunc = processFunc
}

func NewRawMan(configfile string) (*rawMan, error) {
	conf := &rawMan{
		files:    make(chan string, 1024),
		shutdown: make(chan struct{}),
	}

	// default is to look in the executable path
	if len(configfile) <= 0 {
		filename := os.Getenv("HOME")
		configfile = filepath.Join(filepath.Dir(filename), "rawman.config")
	}

	if _, err := toml.DecodeFile(configfile, &conf); err != nil {
		return nil, err
	}

	log.Infof("Read config file '%s'...", configfile)

	var err error
	conf.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	go conf.processQueue()

	return conf, nil
}

func (r *rawMan) Close() {
	if r.watcher != nil {
		r.watcher.Close()
	}
	close(r.shutdown)
}

func (r *rawMan) checkFileForUpdate(filename string) {
	filename = path.Clean(filename)

	if strings.ToLower(filepath.Ext(filename)) == r.InputExt {
		r.files <- filename
	}
}

func (r *rawMan) getRespectiveInputName(outFilename string) (string, error) {
	outFilename, err := filepath.Rel(r.OutputPath, outFilename)
	if err != nil {
		return "", err
	}

	inFilename := filepath.Join(r.InputPath, outFilename)
	extLen := len(r.InputExt)
	if len(inFilename) <= extLen {
		return "", errors.New(fmt.Sprintf("getRespectiveInputName: invalid output filename '%s'", outFilename))
	}

	inFilename = string(inFilename[:len(inFilename)-extLen])

	if len(r.InputExt) > 0 {
		if r.InputExt[0] != '.' {
			inFilename += "."
		}
		inFilename += r.InputExt
	}

	return inFilename, nil
}

func (r *rawMan) getRespectiveOutputName(inFilename string) (string, error) {
	inFilename, err := filepath.Rel(r.InputPath, inFilename)
	if err != nil {
		return "", err
	}

	outFilename := filepath.Join(r.OutputPath, inFilename)
	extLen := len(r.OutputExt)
	if len(outFilename) <= extLen {
		return "", errors.New(fmt.Sprintf("getRespectiveOutputName: invalid output filename '%s'", outFilename))
	}

	outFilename = string(outFilename[:len(outFilename)-extLen])

	if len(r.OutputExt) > 0 {
		if r.OutputExt[0] != '.' {
			outFilename += "."
		}
		outFilename += r.OutputExt
	}

	return outFilename, nil
}

func (r *rawMan) processQueue() {

	var filemap map[string]time.Time = make(map[string]time.Time)
	updateTime := time.Duration(r.HysteresisUpdateTimeMs) * time.Millisecond

	for {
		select {
		case <-time.After(updateTime):
			for filename, t := range filemap {
				diff := time.Since(t)
				if diff > updateTime {
					err := r.processFile(filename)

					if err != nil {
						log.Infof("ProcessFile('%s') failed with error '%s'", filename, err.Error())
					}

					delete(filemap, filename)

					// if there was an error, insert this file again in the channel to try again after a while
					if err != nil {
						r.files <- filename
					}
				}
			}

		case filename := <-r.files:
			filemap[filename] = time.Now()
		}
	}
}

func (r *rawMan) Loop() error {
	log.Println("Walking input path '" + r.InputPath + "'")
	err := filepath.Walk(r.InputPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			err = r.watcher.Watch(path)
			if err != nil {
				return err
			}
		} else {
			r.checkFileForUpdate(path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	log.Println("Walking output path '" + r.OutputPath + "'")
	err = filepath.Walk(r.OutputPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() == false {
			fn, err := r.getRespectiveInputName(path)
			if err != nil {
				return err
			}
			r.checkFileForUpdate(fn)
		} else {
			err := removeDirIfEmpty(path, r.OutputPath)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	for {
		select {
		case ev := <-r.watcher.Event:
			// watch new directories and unwatch deleted ones
			if dir, err := isDir(ev.Name); err == nil && dir {
				if ev.IsDelete() {
					r.watcher.RemoveWatch(ev.Name)
				} else if ev.IsCreate() {
					r.watcher.Watch(ev.Name)
				}
			}
			r.checkFileForUpdate(ev.Name)

		case err := <-r.watcher.Error:
			log.Error("watcher error:", err)
		case <-r.shutdown:
			return nil
		}
	}
}

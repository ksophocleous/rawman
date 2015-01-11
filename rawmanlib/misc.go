package rawmanlib

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func IsDir(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	if fi.IsDir() {
		return true, nil
	}

	return false, nil
}

func GetDirPermissions(permstr string) os.FileMode {
	return getPermissions(permstr, os.FileMode(0775))
}

func GetFilePermissions(permstr string) os.FileMode {
	return getPermissions(permstr, os.FileMode(0664))
}

func getPermissions(permstr string, def os.FileMode) os.FileMode {
	val, err := strconv.ParseInt(permstr, 16, 32)
	if err != nil {
		return def
	}

	// other permissions
	perm := (val & (0x07 << 8)) >> 2

	// group permissions
	perm |= (val & (0x07 << 4)) >> 1

	// owner permissions
	perm |= val & 0x07

	return os.FileMode(perm)
}

// accepts the complete filepath
// removes the output directory if it is empty
// does so recursively
func removeDirIfEmpty(outpath string, stopPath string) error {
	//log.Println("Evaluating possible dir '" + outpath + "'")

	if outpath == stopPath || outpath == "/" {
		return nil
	}

	fi, err := os.Stat(outpath)
	if err != nil {
		if os.IsNotExist(err) {
			// maybe it was a file that is now gone (due to delete of input file)
			return removeDirIfEmpty(filepath.Dir(outpath), stopPath)
			//return nil
		} else {
			return err
		}
	}

	if fi.IsDir() {
		err := filepath.Walk(outpath, func(path string, info os.FileInfo, err error) error {
			if outpath != path {
				return errors.New("not empty")
			}
			return nil
		})

		// if a directory is not empty, stop recursing
		if err != nil {
			return nil
		}

		//log.Println("Removing empty directory '" + outpath + "'")
		// it's empty, try to remove it
		err = os.Remove(outpath)
		if err != nil {
			return errors.New(fmt.Sprintf("removeDirIfEmpty: failed to remove dir '%s'", outpath))
		}
	}

	// do the same thing for the parent dir
	return removeDirIfEmpty(filepath.Dir(outpath), stopPath)
}

// accepts the complete filepath
func EnsureDirPathExists(outpath string, perm os.FileMode) error {
	var path string = filepath.Dir(outpath)

	fi, err := os.Stat(path)
	if err != nil {
		// file not exist is an expected error
		if os.IsNotExist(err) == false {
			return err
		}
	} else {
		// file exists
		if fi.IsDir() == false {
			return errors.New(fmt.Sprintf("directory '%s' is not a directory", path))
		}

		// all good
		return nil
	}

	err = os.MkdirAll(path, perm)
	if err != nil {
		return err
	}

	return err
}

package rawmanlib

import (
	log "github.com/kdar/factorlog"
	"os"
)

func (r *rawMan) processFile(inFilename string) error {
	outFilename, err := r.getRespectiveOutputName(inFilename)
	if err != nil {
		return err
	}

	fi, err := os.Stat(inFilename)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("processing: input file '" + inFilename + "' is gone, deleting output file '" + outFilename + "'")
			err := os.Remove(outFilename)
			if err != nil {
				if os.IsNotExist(err) == false {
					return err
				}
			}
			log.Println("trying to remove base dir of '" + outFilename + "'")
			err = removeDirIfEmpty(outFilename, r.OutputPath)
			return err
		}
		return err
	}

	// input file exists... check timestamp of these input and output file
	mustConvert := false
	foExists := false
	fo, err := os.Stat(outFilename)
	if err != nil {
		if os.IsNotExist(err) {
			mustConvert = true
		} else {
			return err
		}
	} else {
		if fi.ModTime().Sub(fo.ModTime()) > 0 {
			// output file needs updating
			mustConvert = true
			foExists = true
		}
	}

	if mustConvert {
		log.Println("processing: output file '" + outFilename + "' updating")
		// lockedFile := lockFile(outFilename)
		// if lockedFile == false {
		// 	return errors.New(fmt.Sprintf("file '%s' is locked at the moment by another conversion... will retry later", outFilename))
		// }
		// defer unlockFile(outFilename)
		err = ensureDirPathExists(outFilename, getDirPermissions(r.OutputDirMode))
		if err != nil {
			return err
		}

		if r.processFunc != nil {
			err = r.processFunc(inFilename, outFilename)
			if err != nil {
				return err
			}
		}
		// err = convertDng(inFilename, outFilename)
		// if err != nil {
		// 	return err
		// }

		// after the conversion change the access and modify timestamp to be equal to the input file modify timestamp
		err = os.Chtimes(outFilename, fi.ModTime(), fi.ModTime())
		if err != nil {
			return err
		}

		// only change permissions if we create the output file... if the file is already there, leave permissions untouched
		if foExists == false {
			if err = os.Chmod(outFilename, getFilePermissions(r.OutputFileMode)); err != nil {
				log.Println("processing: chmod failed: ", err.Error())
			}
		}
		log.Println("processing: finished processing file '" + outFilename + "'")
	} else {
		log.Println("processing: no change detected for input file '" + inFilename + "'")
	}

	err = removeDirIfEmpty(outFilename, r.OutputPath)
	if err != nil {
		log.Println("removeDirIfEmpty: failed on file '" + outFilename + "'")
	}

	return nil
}

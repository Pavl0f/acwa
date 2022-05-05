// Copyright (c) 2015 HPE Software Inc. All rights reserved.
// Copyright (c) 2013 ActiveState Software Inc. All rights reserved.

package watch

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/fsnotify.v1"
	"gopkg.in/tomb.v1"
)

// InotifyFileWatcher uses inotify to monitor file changes.
type InotifyFileWatcher struct {
	Filename string
	Size     int64
}

func NewInotifyFileWatcher(filename string) *InotifyFileWatcher {
	log.Printf("[CUSTOM] inotify.go NewInotifyFileWatcher")
	fw := &InotifyFileWatcher{filepath.Clean(filename), 0}
	return fw
}

func (fw *InotifyFileWatcher) BlockUntilExists(t *tomb.Tomb) error {
	log.Printf("[CUSTOM] inotify.go BlockUntilExists")
	err := WatchCreate(fw.Filename)
	if err != nil {
		return err
	}
	defer RemoveWatchCreate(fw.Filename)

	// Do a real check now as the file might have been created before
	// calling `WatchFlags` above.
	if _, err = os.Stat(fw.Filename); !os.IsNotExist(err) {
		// file exists, or stat returned an error.
		return err
	}

	events := Events(fw.Filename)

	for {
		log.Printf("[CUSTOM] inotify.go BlockUntilExists loop")
		select {
		case evt, ok := <-events:
			if !ok {
				return fmt.Errorf("inotify watcher has been closed")
			}
			evtName, err := filepath.Abs(evt.Name)
			if err != nil {
				return err
			}
			fwFilename, err := filepath.Abs(fw.Filename)
			if err != nil {
				return err
			}
			if evtName == fwFilename {
				return nil
			}
		case <-t.Dying():
			return tomb.ErrDying
		}
	}
}

func (fw *InotifyFileWatcher) ChangeEvents(t *tomb.Tomb, pos int64) (*FileChanges, error) {
	log.Printf("[CUSTOM] inotify.go ChangeEvents")
	err := Watch(fw.Filename)
	if err != nil {
		return nil, err
	}

	changes := NewFileChanges()
	fw.Size = pos

	go func() {

		events := Events(fw.Filename)

		for {
			log.Printf("[CUSTOM] inotify.go ChangeEvents loop")
			prevSize := fw.Size

			var evt fsnotify.Event
			var ok bool

			select {
			case evt, ok = <-events:
				if !ok {
					log.Printf("[CUSTOM] inotify.go ChangeEvents loop RemoveWatch")
					RemoveWatch(fw.Filename)
					return
				}
			case <-t.Dying():
				log.Printf("[CUSTOM] inotify.go ChangeEvents loop Dying")
				RemoveWatch(fw.Filename)
				return
			}

			switch {
			//With an open fd, unlink(fd) - inotify returns IN_ATTRIB (==fsnotify.Chmod)
			case evt.Op&fsnotify.Chmod == fsnotify.Chmod:
				log.Printf("[CUSTOM] inotify.go ChangeEvents fsnotify.Chmod")
				if _, err := os.Stat(fw.Filename); err != nil {
					if !os.IsNotExist(err) {
						return
					}
				}
				fallthrough

			case evt.Op&fsnotify.Remove == fsnotify.Remove:
				log.Printf("[CUSTOM] inotify.go ChangeEvents fsnotify.Remove")
				fallthrough

			case evt.Op&fsnotify.Rename == fsnotify.Rename:
				log.Printf("[CUSTOM] inotify.go ChangeEvents fsnotify.Rename")
				RemoveWatch(fw.Filename)
				changes.NotifyDeleted()
				return

			case evt.Op&fsnotify.Write == fsnotify.Write:
				log.Printf("[CUSTOM] inotify.go ChangeEvents fsnotify.Write")
				fi, err := os.Stat(fw.Filename)
				if err != nil {
					if os.IsNotExist(err) {
						RemoveWatch(fw.Filename)
						changes.NotifyDeleted()
						return
					}
					log.Printf("E! [logfile] Failed to stat file %v: %v", fw.Filename, err)
				}
				fw.Size = fi.Size()

				if prevSize > 0 && prevSize > fw.Size {
					log.Printf("[CUSTOM] inotify.go ChangeEvents NotifyTruncated")
					changes.NotifyTruncated()
				} else {
					log.Printf("[CUSTOM] inotify.go ChangeEvents NotifyModified")
					changes.NotifyModified()
				}
				prevSize = fw.Size
			}
		}
	}()

	return changes, nil
}

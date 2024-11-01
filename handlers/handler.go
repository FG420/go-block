package handlers

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dgraph-io/badger"
)

const (
	DbPath      = "./tmp/blocks_%s"
	GenesisData = "First Transaction from Genesis"
)

func DbExist(path string) bool {
	if _, err := os.Stat(path + "/MANIFEST"); os.IsNotExist(err) {
		return false
	}
	return true
}

func Retry(dir string, originalOpts badger.Options) (*badger.DB, error) {
	lockPath := filepath.Join(dir, "LOCK")
	if err := os.Remove(lockPath); err != nil {
		return nil, fmt.Errorf(`Removing "LOCK": %s`, err)
	}

	retryOpts := originalOpts
	retryOpts.Truncate = true
	db, err := badger.Open(retryOpts)
	return db, err
}

func OpenDB(dir string, opts badger.Options) (*badger.DB, error) {
	if db, err := badger.Open(opts); err != nil {
		if strings.Contains(err.Error(), "LOCK") {
			if db, err := Retry(dir, opts); err == nil {
				log.Println("Database unlocked, value log truncated")
				return db, nil
			}
			log.Println("Could not unlock db: ", err)
		}
		return nil, err
	} else {
		return db, nil
	}
}

func HandleErr(err error) {
	if err != nil {
		log.Panic("ERROR: ", err)
	}
}

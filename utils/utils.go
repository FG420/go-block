package utils

import (
	"log"
	"os"
)

const (
	DbPath      = "./tmp/blocks"
	DbFile      = "./tmp/blocks/MANIFEST"
	GenesisData = "First Transaction from Genesis"
)

func DbExist() bool {
	if _, err := os.Stat(DbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

func HandleErr(err error) {
	if err != nil {
		log.Panic("Error: ", err)
	}
}

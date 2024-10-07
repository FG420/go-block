package wallet

import (
	"github.com/FG420/go-block/utils"
	"github.com/mr-tron/base58"
)

func Base58Encode(input []byte) []byte {
	encode := base58.Encode(input)
	return []byte(encode)
}

func Base58Decode(input []byte) []byte {
	decode, err := base58.Decode(string(input[:]))
	utils.HandleErr(err)
	return decode
}

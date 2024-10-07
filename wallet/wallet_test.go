package wallet_test

import (
	"log"
	"testing"

	"github.com/FG420/go-block/wallet"
)

func TestWallet(t *testing.T) {
	wallet := wallet.MakeWallet()

	log.Println(wallet)
}

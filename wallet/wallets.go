package wallet

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

const walletFile = "./tmp/wallets_%s.json"

// const walletFile = "./tmp/wallets_%s.data"

type Wallets struct {
	Wallets map[string]*Wallet
}

func CreateWallets(nodeId string) (*Wallets, error) {
	var wallets Wallets

	wallets.Wallets = make(map[string]*Wallet)
	err := wallets.LoadFile(nodeId)

	return &wallets, err
}

func (ws *Wallets) GetAddress(addr string) *Wallet {
	return ws.Wallets[addr]
}

func (ws *Wallets) GetAllAddresses() []string {
	var addrs []string

	for addr := range ws.Wallets {
		addrs = append(addrs, addr)
	}

	return addrs
}

func (ws *Wallets) AddWallet() string {
	wallet := MakeWallet()
	addr := fmt.Sprintf("%s", wallet.Address())

	ws.Wallets[addr] = wallet
	return addr
}

// Json Save & Load File
func (ws *Wallets) LoadFile(nodeId string) error {
	walletFile := fmt.Sprintf(walletFile, nodeId)
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return nil
	}

	fileContent, err := os.ReadFile(walletFile)
	if err != nil {
		log.Panic(err)
	}

	var temp struct {
		Wallets map[string]*Wallet `json:"Wallets"`
	}

	err = json.Unmarshal(fileContent, &temp)
	if err != nil {
		log.Panic(err)
	}

	ws.Wallets = temp.Wallets

	return nil
}

func (ws Wallets) SaveFile(nodeId string) {
	walletFile := fmt.Sprintf(walletFile, nodeId)
	jsonData, err := json.Marshal(ws)
	if err != nil {
		log.Panic(err)
	}

	err = os.WriteFile(walletFile, jsonData, 0666)
	if err != nil {
		log.Panic(err)
	}
}

// Bog Save & Load File
// func (ws *Wallets) LoadFile(nodeId string) error {
// 	walletFile := fmt.Sprintf(walletFile, nodeId)
// 	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
// 		return nil
// 	}

// 	var wallets Wallets
// 	fileContent, err := os.ReadFile(walletFile)
// 	handlers.HandleErr(err)

// 	gob.Register(elliptic.P256())
// 	dec := gob.NewDecoder(bytes.NewReader(fileContent))
// 	err = dec.Decode(&wallets)
// 	handlers.HandleErr(err)

// 	ws.Wallets = wallets.Wallets
// 	return nil
// }

// func (ws Wallets) SaveFile(nodeId string) {
// 	var content bytes.Buffer
// 	walletFile := fmt.Sprintf(walletFile, nodeId)

// 	gob.Register(elliptic.P256())

// 	enc := gob.NewEncoder(&content)
// 	err := enc.Encode(ws)
// 	handlers.HandleErr(err)

// 	err = os.WriteFile(walletFile, content.Bytes(), 0644)
// 	handlers.HandleErr(err)
// }

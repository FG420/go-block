package network

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"syscall"

	"github.com/vrecan/death/v3"

	"github.com/FG420/go-block/blockchain"
	"github.com/FG420/go-block/handlers"
)

const (
	protocol  = "tcp"
	version   = 1
	cmdLength = 12
)

var (
	nodeAddr        string
	minerAddr       string
	KnownNodes      = []string{"localhost:3000"}
	blocksInTransit = [][]byte{}
	memoryPool      = make(map[string]blockchain.Transaction)
)

type (
	Addr struct {
		AddrList []string
	}

	Block struct {
		AddrFrom string
		Block    []byte
	}

	GetBlocks struct {
		AddrFrom string
	}

	GetData struct {
		ID       []byte
		AddrFrom string
		Type     string
	}

	Inv struct {
		AddrFrom string
		Type     string
		Items    [][]byte
	}

	Tx struct {
		AddrFrom    string
		Transaction []byte
	}

	Version struct {
		Version    int
		BestHeight int
		AddrFrom   string
	}
)

func CmdToBytes(cmd string) []byte {
	var bytes [cmdLength]byte

	for i, c := range cmd {
		bytes[i] = byte(c)
	}
	return bytes[:]
}

func BytesToCmd(bytes []byte) string {
	var cmd []byte

	for _, b := range bytes {
		if b != 0x0 {
			cmd = append(cmd, b)
		}
	}
	return fmt.Sprintf("%s", cmd)
}

func CloseDB(chain *blockchain.BlockChain) {
	d := death.NewDeath(syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	d.WaitForDeathWithFunc(func() {
		defer os.Exit(1)
		defer runtime.Goexit()
		chain.Database.Close()
	})
}

func GobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	handlers.HandleErr(err)

	return buff.Bytes()
}

func RequestBlocks() {
	for _, node := range KnownNodes {
		SendGetBlocks(node)
	}
}

func NodeIsKnown(addr string) bool {
	for _, node := range KnownNodes {
		if node == addr {
			return true
		}
	}

	return false
}

func HandleConnection(conn net.Conn, chain *blockchain.BlockChain) {
	req, err := io.ReadAll(conn)
	defer conn.Close()
	handlers.HandleErr(err)

	cmd := BytesToCmd(req[:cmdLength])
	fmt.Printf("Received %s command\n", cmd)

	switch cmd {
	case "addr":
		HandleAddr(req)
	case "block":
		HandleBlock(req, chain)
	case "inv":
		HandleInv(req, chain)
	case "getblocks":
		HandleGetBlocks(req, chain)
	case "getdata":
		HandleGetData(req, chain)
	case "tx":
		HandleTx(req, chain)
	case "version":
		HandleVersion(req, chain)
	default:
		fmt.Println("Unknown Command")
	}
}

func StartServer(nodeId, mineraddr string) {
	nodeAddr = fmt.Sprintf("localhost:%s", nodeId)
	minerAddr = mineraddr

	ln, err := net.Listen(protocol, nodeAddr)
	handlers.HandleErr(err)
	defer ln.Close()

	chain := blockchain.ContinueBlockChain(nodeId)
	defer chain.Database.Close()
	go CloseDB(chain)

	if nodeAddr != KnownNodes[0] {
		SendVersion(KnownNodes[0], chain)
	}

	for {
		conn, err := ln.Accept()
		handlers.HandleErr(err)
		go HandleConnection(conn, chain)
	}
}

func HandleAddr(req []byte) {
	var buff bytes.Buffer
	var payload Addr

	buff.Write(req[cmdLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	handlers.HandleErr(err)

	KnownNodes = append(KnownNodes, payload.AddrList...)
	fmt.Printf("There are %d known nodes\n", len(KnownNodes))
	RequestBlocks()
}

func HandleBlock(req []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Block

	buff.Write(req[cmdLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	handlers.HandleErr(err)

	blockData := payload.Block
	block := blockchain.Deserialize(blockData)

	fmt.Println("Received a new block.")

	chain.AddBlock(block)

	fmt.Printf("Added block %x\n", block.Hash)

	if len(blocksInTransit) > 0 {
		blockHash := blocksInTransit[0]
		SendGetData(payload.AddrFrom, "block", blockHash)

		blocksInTransit = blocksInTransit[1:]
	} else {
		utxoSet := blockchain.UTXOSet{BlockChain: chain}
		utxoSet.Reindex()
	}
}

func HandleGetBlocks(req []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload GetBlocks

	buff.Write(req[cmdLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	handlers.HandleErr(err)

	blocks := chain.GetBlockHashes()
	SendInv(payload.AddrFrom, "block", blocks)
}

func HandleGetData(req []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload GetData

	buff.Write(req[cmdLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	handlers.HandleErr(err)

	if payload.Type == "block" {
		block, err := chain.GetBlock([]byte(payload.ID))
		if err != nil {
			return
		}
		SendBlock(payload.AddrFrom, block)
	}

	if payload.Type == "tx" {
		txId := hex.EncodeToString(payload.ID)
		tx := memoryPool[txId]

		SendTx(payload.AddrFrom, &tx)
	}
}

func HandleVersion(req []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Version

	buff.Write(req[cmdLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	handlers.HandleErr(err)

	bestHeight := chain.GetBestHeight()
	otherHeight := payload.BestHeight

	if bestHeight < otherHeight {
		SendGetBlocks(payload.AddrFrom)
	} else if bestHeight > otherHeight {
		SendVersion(payload.AddrFrom, chain)
	}

	if !NodeIsKnown(payload.AddrFrom) {
		KnownNodes = append(KnownNodes, payload.AddrFrom)
	}
}

func HandleTx(req []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Tx

	buff.Write(req[cmdLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	handlers.HandleErr(err)

	txData := payload.Transaction
	tx := blockchain.DeserializeTransaction(txData)
	memoryPool[hex.EncodeToString(tx.ID)] = tx

	fmt.Printf("%s, %d", nodeAddr, len(memoryPool))

	if nodeAddr == KnownNodes[0] {
		for _, node := range KnownNodes {
			if node != nodeAddr && node != payload.AddrFrom {
				SendInv(node, "tx", [][]byte{tx.ID})
			}
		}
	} else {
		if len(memoryPool) >= 2 && len(minerAddr) > 0 {
			MineTx(chain)
		}
	}
}

func MineTx(chain *blockchain.BlockChain) {
	var txs []*blockchain.Transaction

	for id := range memoryPool {
		fmt.Printf("tx: %s\n", memoryPool[id].ID)
		tx := memoryPool[id]
		if chain.VerifyTransaction(&tx) {
			txs = append(txs, &tx)
		}
	}

	if len(txs) == 0 {
		fmt.Println("All Transactions are invalid")
		return
	}

	cbTx := blockchain.CoinbaseTx(minerAddr, "")
	txs = append(txs, cbTx)

	newBlock := chain.MineBlock(txs)
	utxoSet := blockchain.UTXOSet{BlockChain: chain}
	utxoSet.Reindex()

	fmt.Println("New Block mined")

	for _, tx := range txs {
		txID := hex.EncodeToString(tx.ID)
		delete(memoryPool, txID)
	}

	for _, node := range KnownNodes {
		if node != nodeAddr {
			SendInv(node, "block", [][]byte{newBlock.Hash})
		}
	}

	if len(memoryPool) > 0 {
		MineTx(chain)
	}
}

func HandleInv(req []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Inv

	buff.Write(req[cmdLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	handlers.HandleErr(err)

	fmt.Printf("Received inventory with %d %s\n", len(payload.Items), payload.Type)

	if payload.Type == "block" {
		blocksInTransit = payload.Items

		blockHash := payload.Items[0]
		SendGetData(payload.AddrFrom, "block", blockHash)

		newInTransit := [][]byte{}
		for _, b := range blocksInTransit {
			if bytes.Compare(b, blockHash) != 0 {
				newInTransit = append(newInTransit, b)
			}
		}
		blocksInTransit = newInTransit
	}

	if payload.Type == "tx" {
		txID := payload.Items[0]

		if memoryPool[hex.EncodeToString(txID)].ID == nil {
			SendGetData(payload.AddrFrom, "tx", txID)
		}
	}
}

func ExtractCmd(req []byte) []byte {
	return req[:cmdLength]
}

func SendData(addr string, data []byte) {
	conn, err := net.Dial(protocol, addr)

	if err != nil {
		fmt.Printf("%s isn't available\n", addr)

		var updatedNodes []string

		for _, node := range KnownNodes {
			if node != addr {
				updatedNodes = append(updatedNodes, node)
			}
		}

		KnownNodes = updatedNodes
		return
	}

	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))
	handlers.HandleErr(err)
}

func SendAddr(addr string) {
	nodes := Addr{KnownNodes}
	nodes.AddrList = append(nodes.AddrList, nodeAddr)
	payload := GobEncode(nodes)
	req := append(CmdToBytes("addr"), payload...)

	SendData(addr, req)
}

func SendBlock(addr string, b *blockchain.Block) {
	data := Block{nodeAddr, b.Serialize()}
	payload := GobEncode(data)
	req := append(CmdToBytes("block"), payload...)

	SendData(addr, req)
}

func SendInv(addr, kind string, items [][]byte) {
	invetory := Inv{nodeAddr, kind, items}
	payload := GobEncode(invetory)
	req := append(CmdToBytes("inv"), payload...)

	SendData(addr, req)
}

func SendTx(addr string, tnx *blockchain.Transaction) {
	data := Tx{nodeAddr, tnx.Serialize()}
	payload := GobEncode(data)
	req := append(CmdToBytes("tx"), payload...)

	SendData(addr, req)
}

func SendVersion(addr string, chain *blockchain.BlockChain) {
	bestHeight := chain.GetBestHeight()
	vers := Version{version, bestHeight, nodeAddr}
	payload := GobEncode(vers)
	req := append(CmdToBytes("version"), payload...)

	SendData(addr, req)

}

func SendGetBlocks(addr string) {
	payload := GobEncode(GetBlocks{nodeAddr})
	req := append(CmdToBytes("getblocks"), payload...)

	SendData(addr, req)
}

func SendGetData(addr, kind string, id []byte) {
	payload := GobEncode(GetData{id, kind, nodeAddr})
	req := append(CmdToBytes("getdata"), payload...)

	SendData(addr, req)
}

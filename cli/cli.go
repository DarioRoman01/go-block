package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"

	"github.com/Haizza1/go-block/blockchain"
	"github.com/Haizza1/go-block/network"
	"github.com/Haizza1/go-block/wallet"
)

type CommandLine struct{}

// print usage will print the cli usage
func (cli *CommandLine) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("	getbalance -address <ADDRESS> - get the balance for the given address")
	fmt.Println(" 	createBlockchain -address <ADDRESS> create a blockchain with the given address")
	fmt.Println(" 	printchain - Prints the blocks in the Blockchain")
	fmt.Println(" 	send -from <FROM> -to <TO> -amount <AMOUNT> -mine - Send Send amount of coins")
	fmt.Println("	createWallet - creates a new Wallet")
	fmt.Println("	listaddresses - list the address in our wallet file")
	fmt.Println("	reindex - Rebuilds The unspent transactions outputs set")
	fmt.Println(" 	startnode -miner ADDRESS - Start a node with ID specified in NODE_ID env. var. -miner enables mining")
}

// validateArgs will check if args were given
func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

// validateAddress will check if the given address is valid
func (cli *CommandLine) validateAddress(address string) {
	if !wallet.ValidateAddress(address) {
		fmt.Printf("Adress %s is invalid\n", address)
		runtime.Goexit()
	}
}

// Start node will start the node in the blockchain network
func (cli *CommandLine) StartNode(nodeID, minerAddress string) {
	fmt.Printf("Starting Node... %s\n", nodeID)

	if len(minerAddress) > 0 {
		if wallet.ValidateAddress(minerAddress) {
			fmt.Println("Mining is on. Address to recieve rewards: ", minerAddress)
		} else {
			fmt.Println("Wrong Miner address!")
			runtime.Goexit()
		}
	}

	network.StartServer(nodeID, minerAddress)
}

// reindex unspent transactions will call the reindex method
// on the UtxoSet
func (cli *CommandLine) reindexUTXO(nodeID string) {
	chain := blockchain.ContinueBlockChain(nodeID)
	defer chain.Database.Close()
	UTXSet := blockchain.UTXOSet{BlockChain: chain}
	UTXSet.Reindex()

	count := UTXSet.CountTransactions()
	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", count)
}

// listAddresses will print all the addresses in the wallet.data file
func (cli *CommandLine) listAddresses(nodeId string) {
	wallets, _ := wallet.CreateWallets(nodeId)
	addresses := wallets.GetAllAddress()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

// createWallet will create new wallet and saved to the wallet file
func (cli *CommandLine) createWallet(nodeID string) {
	wallets, _ := wallet.CreateWallets(nodeID)
	address := wallets.AddWallet()
	wallets.SaveFile(nodeID)
	fmt.Printf("New address is: %s\n", address)
}

// printChain will print all the blocks in the blockchain
func (cli *CommandLine) printChain(nodeID string) {
	chain := blockchain.ContinueBlockChain(nodeID)
	defer chain.Database.Close()
	iter := chain.Iterator()

	for {
		block := iter.Next()

		fmt.Printf("Previos Hash: %x\n", block.PrevHash)
		fmt.Printf("Block Hash: %x\n", block.Hash)

		pow := blockchain.NewProof(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}

		fmt.Println()

		if len(block.PrevHash) == 0 {
			break
		}
	}
}

// create blockchain will create a new blockchain instance with the given address
func (cli *CommandLine) createBLockChain(address, nodeID string) {
	cli.validateAddress(address)
	chain := blockchain.InitBLockChain(address, nodeID)
	chain.Database.Close()
	fmt.Println("Finished!")
}

func (cli *CommandLine) getBalance(address, nodeID string) {
	cli.validateAddress(address)
	chain := blockchain.ContinueBlockChain(nodeID)
	UTXIOSet := blockchain.UTXOSet{BlockChain: chain}
	defer chain.Database.Close()

	balance := 0
	pubKeyHash := wallet.Base58Encode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	unspentTxs := UTXIOSet.FindUTXO(pubKeyHash)

	for _, out := range unspentTxs {
		balance += out.Value
	}

	fmt.Printf("Balance of %s: %d\n", address, balance)
}

func (cli *CommandLine) send(from, to string, amount int, nodeID string, mineNow bool) {
	cli.validateAddress(from)
	cli.validateAddress(to)

	chain := blockchain.ContinueBlockChain(nodeID)
	UTXIOSet := &blockchain.UTXOSet{BlockChain: chain}
	defer chain.Database.Close()

	wallets, err := wallet.CreateWallets(nodeID)
	if err != nil {
		log.Println(err)
		runtime.Goexit()
	}

	wallet := wallets.GetWallet(from)

	tx := blockchain.NewTransaction(&wallet, to, amount, UTXIOSet)
	if mineNow {
		cbTx := blockchain.CoinbaseTx(from, "")
		txs := []*blockchain.Transaction{cbTx, tx}
		block := chain.MineBlock(txs)
		UTXIOSet.Update(block)
	} else {
		network.SendTx(network.KnownNodes[0], tx)
		fmt.Println("Sending transaction....")
	}

	fmt.Println("Success!")
}

// Run will start the comman line app and validate the args
func (cli *CommandLine) Run() {
	cli.validateArgs()

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Println("NODE_ID env is not set!!")
		runtime.Goexit()
	}

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBLockchainCmd := flag.NewFlagSet("createBlockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createWallet", flag.ExitOnError)
	reindexCmd := flag.NewFlagSet("reindex", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startNode", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBLockChainAddress := createBLockchainCmd.String("address", "", "The address to send genesis block reward to")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")
	sendMineNow := sendCmd.Bool("mine", false, "Mine immediatly on the same node")
	startNodeMiner := startNodeCmd.String("miner", "", "Enable mining node and send reward to")

	switch os.Args[1] {
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		blockchain.CheckError(err)

	case "send":
		err := sendCmd.Parse(os.Args[2:])
		blockchain.CheckError(err)

	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		blockchain.CheckError(err)

	case "createBlockchain":
		err := createBLockchainCmd.Parse(os.Args[2:])
		blockchain.CheckError(err)

	case "startNode":
		err := startNodeCmd.Parse(os.Args[2:])
		blockchain.CheckError(err)

	case "createWallet":
		err := createWalletCmd.Parse(os.Args[2:])
		blockchain.CheckError(err)

	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		blockchain.CheckError(err)

	case "reindex":
		err := reindexCmd.Parse(os.Args[2:])
		blockchain.CheckError(err)

	default:
		cli.printUsage()
		runtime.Goexit()
	}

	if reindexCmd.Parsed() {
		cli.reindexUTXO(nodeID)
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			cli.printUsage()
			runtime.Goexit()
		}

		cli.getBalance(*getBalanceAddress, nodeID)
	}

	if startNodeCmd.Parsed() {
		if *startNodeMiner == "" {
			cli.printUsage()
			runtime.Goexit()
		}

		cli.StartNode(nodeID, *startNodeMiner)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			cli.printUsage()
			runtime.Goexit()
		}

		cli.send(*sendFrom, *sendTo, *sendAmount, nodeID, *sendMineNow)
	}

	if printChainCmd.Parsed() {
		cli.printChain(nodeID)
	}

	if createWalletCmd.Parsed() {
		cli.createWallet(nodeID)
	}

	if listAddressesCmd.Parsed() {
		cli.listAddresses(nodeID)
	}

	if createBLockchainCmd.Parsed() {
		if *createBLockChainAddress == "" {
			cli.printUsage()
			runtime.Goexit()
		}

		cli.createBLockChain(*createBLockChainAddress, nodeID)
	}
}

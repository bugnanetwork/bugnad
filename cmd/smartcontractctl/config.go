package main

import (
	"os"
	"path/filepath"

	"github.com/bugnanetwork/bugnad/infrastructure/config"
	"github.com/bugnanetwork/bugnad/util"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
)

const (
	defaultRPCServer      = "localhost"
	defaultLogFilename    = "bugnasc.log"
	defaultErrLogFilename = "bugnasc_err.log"
)

const (
	deploySubCmd        = "deploy"
	callContractSubCmd  = "callcontract"
	evmAddrSubCmd       = "evmaddr"
	queryContractSubCmd = "querycontract"
)

var (
	defaultAppDir     = util.AppDir("bugnasc", false)
	defaultLogFile    = filepath.Join(defaultAppDir, defaultLogFilename)
	defaultErrLogFile = filepath.Join(defaultAppDir, defaultErrLogFilename)
)

type configFlags struct {
	RPCServer  string `short:"s" long:"rpcserver" description:"RPC server to connect to"`
	PrivateKey string `short:"p" long:"privatekey" description:"Private key to use for signing"`
	config.NetworkFlags
}

type deployConfig struct {
	ContractCode string `short:"c" long:"contractcode" description:"Path to the contract code"`
	configFlags
}

type callContractConfig struct {
	ContractAddress string   `short:"a" long:"contractaddress" description:"Address of the contract"`
	AbiFile         string   `short:"b" long:"abifile" description:"Path to the ABI file"`
	Method          string   `short:"m" long:"method" description:"Method to call"`
	Args            []string `short:"g" long:"args" description:"Arguments to the method"`
	RawInput        string   `short:"i" long:"rawinput" description:"Raw input data"`
	configFlags
}

type evmAddrConfig struct {
	BugnaAddress string `short:"a" long:"bugnaaddress" description:"Bugna address"`
	config.NetworkFlags
}

type queryContractConfig struct {
	RPCServer       string   `short:"s" long:"rpcserver" description:"RPC server to connect to"`
	ContractAddress string   `short:"a" long:"contractaddress" description:"Address of the contract"`
	AbiFile         string   `short:"b" long:"abifile" description:"Path to the ABI file"`
	Method          string   `short:"m" long:"method" description:"Method to call"`
	Args            []string `short:"g" long:"args" description:"Arguments to the method"`
	config.NetworkFlags
}

func parseCommandLine() (subCommand string, config interface{}) {
	cfg := &configFlags{}
	parser := flags.NewParser(cfg, flags.PrintErrors|flags.HelpFlag)

	initLog(defaultLogFile, defaultErrLogFile)

	deployConf := &deployConfig{}
	parser.AddCommand(deploySubCmd, "Deploy a contract", "Deploys a contract to the network", deployConf)

	callContractConf := &callContractConfig{}
	parser.AddCommand(callContractSubCmd, "Call a contract", "Calls a method on a contract", callContractConf)

	evmAddrConf := &evmAddrConfig{}
	parser.AddCommand(evmAddrSubCmd, "Convert Bugna address to EVM address", "Converts a Bugna address to an EVM address", evmAddrConf)

	queryContractConf := &queryContractConfig{}
	parser.AddCommand(queryContractSubCmd, "Query a contract", "Queries a contract on the network", queryContractConf)

	_, err := parser.Parse()
	if err != nil {
		var flagsErr *flags.Error
		if ok := errors.As(err, &flagsErr); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
		return "", nil
	}

	switch parser.Command.Active.Name {
	case deploySubCmd:
		combineNetworkFlags(&deployConf.NetworkFlags, &cfg.NetworkFlags)
		err := deployConf.ResolveNetwork(parser)
		if err != nil {
			printErrorAndExit(err.Error())
		}
		validateDeployConfig(deployConf)
		config = deployConf
	case callContractSubCmd:
		combineNetworkFlags(&callContractConf.NetworkFlags, &cfg.NetworkFlags)
		err := callContractConf.ResolveNetwork(parser)
		if err != nil {
			printErrorAndExit(err.Error())
		}
		validateCallContractConfig(callContractConf)
		config = callContractConf
	case evmAddrSubCmd:
		combineNetworkFlags(&evmAddrConf.NetworkFlags, &cfg.NetworkFlags)
		err := evmAddrConf.ResolveNetwork(parser)
		if err != nil {
			printErrorAndExit(err.Error())
		}
		validateEVMAddrConfig(evmAddrConf)
		config = evmAddrConf
	case queryContractSubCmd:
		combineNetworkFlags(&queryContractConf.NetworkFlags, &cfg.NetworkFlags)
		err := queryContractConf.ResolveNetwork(parser)
		if err != nil {
			printErrorAndExit(err.Error())
		}
		validateQueryContractConfig(queryContractConf)
		config = queryContractConf
	}

	return parser.Command.Active.Name, config
}

func combineNetworkFlags(dst, src *config.NetworkFlags) {
	dst.Testnet = dst.Testnet || src.Testnet
	dst.Simnet = dst.Simnet || src.Simnet
	dst.Devnet = dst.Devnet || src.Devnet
	if dst.OverrideDAGParamsFile == "" {
		dst.OverrideDAGParamsFile = src.OverrideDAGParamsFile
	}
}

func validateDeployConfig(cfg *deployConfig) {
	if cfg.ContractCode == "" {
		printErrorAndExit("contract code is required")
	}

	if cfg.PrivateKey == "" {
		printErrorAndExit("private key is required")
	}
}

func validateCallContractConfig(cfg *callContractConfig) {
	if cfg.ContractAddress == "" {
		printErrorAndExit("contract address is required")
	}

	if cfg.PrivateKey == "" {
		printErrorAndExit("private key is required")
	}

	if cfg.RawInput == "" {
		if cfg.Method == "" {
			printErrorAndExit("method is required")
		}

		if cfg.AbiFile == "" {
			printErrorAndExit("ABI file is required")
		}
	}
}

func validateEVMAddrConfig(cfg *evmAddrConfig) {
	if cfg.BugnaAddress == "" {
		printErrorAndExit("Bugna address is required")
	}
}

func validateQueryContractConfig(cfg *queryContractConfig) {
	if cfg.ContractAddress == "" {
		printErrorAndExit("contract address is required")
	}

	if cfg.Method == "" {
		printErrorAndExit("method is required")
	}

	if cfg.AbiFile == "" {
		printErrorAndExit("ABI file is required")
	}
}

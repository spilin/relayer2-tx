package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configFile, databaseURL, rpcUrl, blockFile string
var debug bool
var fromBlock, toBlock uint64

type Transaction struct {
	Hash string `json:"hash"`
}

type Block struct {
	Number       string        `json:"number"`
	Transactions []Transaction `json:"transactions"`
}

func main() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file (default is config/local.yaml)")
	rootCmd.PersistentFlags().StringVarP(&blockFile, "blockFile", "b", "", "block file (default is config/local.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debugging(default false)")
	rootCmd.PersistentFlags().StringVar(&databaseURL, "database", "", "database url (default postgres://aurora:aurora@database/aurora)")
	rootCmd.PersistentFlags().Uint64VarP(&fromBlock, "fromBlock", "f", 0, "block to start from. Ignored if missing or 0. (default 0)")
	rootCmd.PersistentFlags().Uint64VarP(&toBlock, "toBlock", "t", 0, "block to end on. Ignored if missing or 0. (default 0)")
	rootCmd.PersistentFlags().StringVarP(&rpcUrl, "rpc", "r", "", "rpc url")
	cobra.CheckErr(rootCmd.Execute())
}

func initConfig() {
	if configFile != "" {
		log.Warn().Msg(fmt.Sprint("Using config file:", viper.ConfigFileUsed()))
		viper.SetConfigFile(configFile)
	} else {
		viper.AddConfigPath("config")
		viper.AddConfigPath("../../config")
		viper.SetConfigName("local")
		if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
			panic(fmt.Errorf("Flags are not bindable: %v\n", err))
		}
	}
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err == nil {
		log.Warn().Msg(fmt.Sprint("Using config file:", viper.ConfigFileUsed()))
	}

	debug = viper.GetBool("debug")
	databaseURL = viper.GetString("database")
	rpcUrl = viper.GetString("rpc")
	blockFile = viper.GetString("blockFile")
	fromBlock = viper.GetUint64("fromBlock")
	toBlock = viper.GetUint64("toBlock")
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

var rootCmd = &cobra.Command{
	Use:     "indexer",
	Short:   "Imports verified contracts info to blockscout from aurorascan.",
	Long:    "Imports verified contracts info to blockscout from aurorascan.",
	Version: "0.0.1",
	Run: func(cmd *cobra.Command, args []string) {
		pgpool, err := pgxpool.Connect(context.Background(), databaseURL)
		if err != nil {
			panic(fmt.Errorf("unable to connect to database %s: %v", databaseURL, err))
		}
		defer pgpool.Close()

		client, err := rpc.DialContext(context.Background(), rpcUrl)
		if err != nil {
			panic(fmt.Errorf("Unable to connect to %s: %v\n", rpcUrl, err))
		}
		defer client.Close()

		seq, err := os.ReadFile(blockFile)
		if err == nil {
			seqUint64, err := strconv.ParseUint(string(seq[:]), 10, 64)
			if err == nil {
				fromBlock = seqUint64
			}
		}

		file, _ := os.OpenFile(blockFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		defer file.Close()

		interrupt := make(chan os.Signal, 10)
		signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGINT)
		go follow(pgpool, client, file)

		select {
		case <-interrupt:
			os.Exit(0)
		}

	},
}

func follow(pgpool *pgxpool.Pool, client *rpc.Client, file *os.File) {
	var resp Block
	for {
		if err := client.Call(&resp, "eth_getBlockByNumber", fromBlock, true); err != nil {
			panic(err)
		}

		if resp.Number == "" {
			insertMissingBlock(pgpool, fmt.Sprintf("('%v')", fromBlock))
		}
		for _, tx := range resp.Transactions {
			insertTx(pgpool, fmt.Sprintf("('%s')", tx.Hash))
		}

		file.Truncate(0)
		file.Seek(0, 0)
		file.WriteString(strconv.FormatUint(fromBlock, 10))
		fromBlock += 1
		if (toBlock > 0) && (toBlock <= fromBlock) {
			fmt.Printf("Ended on %v\n", fromBlock)
			os.Exit(0)
		}
	}
}

func insertTx(pgpool *pgxpool.Pool, value string) {
	insertSql := fmt.Sprintf(`
	INSERT INTO relayer2_tx (tx)
    VALUES %s
    ON CONFLICT (tx) DO UPDATE SET count = relayer2_tx.count + 1;`, value)

	if _, err := pgpool.Exec(context.Background(), insertSql); err != nil {
		panic(err)
	}
}

func insertMissingBlock(pgpool *pgxpool.Pool, block string) {
	insertSql := fmt.Sprintf(`
	INSERT INTO gaps (block) VALUES %s ON CONFLICT (block) DO NOTHING`, block)

	if _, err := pgpool.Exec(context.Background(), insertSql); err != nil {
		panic(err)
	}
}

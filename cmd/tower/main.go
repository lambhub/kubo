package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/ipfs/kubo/tower"
	"github.com/spf13/cobra"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	os.Exit(mainRet())
}

const RED = "\033[0;31m"
const NC = "\033[0m"

func mainRet() int {
	sign := make(chan os.Signal, 1)
	signal.Notify(sign, os.Interrupt, syscall.SIGTERM)
	ctx, cancelFunc := context.WithCancel(context.Background())
	var (
		remoteUrl    string
		chainId      int64
		duration     time.Duration
		duaStr       string
		privateKey   string
		contractAddr string
	)

	rootCmd := &cobra.Command{
		Use: "tower",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			_, err := url.Parse(remoteUrl)
			if err != nil {
				return err
			}
			duration, err = time.ParseDuration(duaStr)
			if err != nil {
				return err
			}
			if len(privateKey) == 0 {
				return errors.New("private key must not be none")
			}
			if len(contractAddr) == 0 {
				return errors.New("contractAddr must not be none")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := tower.NewTower(ctx, remoteUrl, duration, chainId, privateKey, contractAddr)
			if err != nil {
				return err
			}
			go t.Run()
			listenAndWait(sign, cancelFunc)
			return nil
		},
	}
	rootCmd.Flags().StringVar(&remoteUrl, "u", "http://18.143.13.243:8545", "web3 url")
	rootCmd.Flags().Int64VarP(&chainId, "cid", "c", 92001, "chainId")
	rootCmd.Flags().StringVar(&duaStr, "d", "1m", "duration")
	rootCmd.Flags().StringVar(&privateKey, "pk", "", "hex format private key")
	rootCmd.Flags().StringVar(&contractAddr, "ca", "", "hex format contract address")
	if err := rootCmd.Execute(); err != nil {
		fmt.Print(RED)
		fmt.Print(err)
		fmt.Println(NC)
		return 1
	}
	return 0
}

func listenAndWait(c chan os.Signal, f func()) {
	<-c
	f()
	fmt.Println("shutdown")
}

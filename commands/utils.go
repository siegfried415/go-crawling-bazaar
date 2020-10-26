
package commands

import (
	"context" 
        "fmt"
	"time" 
	"errors" 
        "io"

	//wyong, 20201022
	//"sync/atomic"
	"os" 
	"bufio" 
	"strings" 
	"syscall" 

	//wyong, 20200824 
	url "net/url" 	

	"golang.org/x/crypto/ssh/terminal"

	//wyong, 20201007 
	host "github.com/libp2p/go-libp2p-host"
	//protocol "github.com/libp2p/go-libp2p-core/protocol" 

        "github.com/siegfried415/gdf-rebuild/crypto/hash"

        //"github.com/siegfried415/go-crawling-market/types"
        pi "github.com/siegfried415/gdf-rebuild/presbyterian/interfaces"
        client "github.com/siegfried415/gdf-rebuild/client"

)

var (
        waitTxConfirmationMaxDuration time.Duration = 1 * time.Minute 
)


// PrintString prints a given Stringer to the writer.
func PrintString(w io.Writer, s fmt.Stringer) error {
        _, err := fmt.Fprintln(w, s.String())
        return err
}

func wait(host host.Host, txHash hash.Hash) (err error) {
        var ctx, cancel = context.WithTimeout(context.Background(), waitTxConfirmationMaxDuration)
        defer cancel()
        var state pi.TransactionState
        state, err = client.WaitTxConfirmation(ctx, host, txHash)
        //ConsoleLog.WithFields(logrus.Fields{
        //        "tx_hash":  txHash,
        //        "tx_state": state,
        //}).WithError(err).Info("wait transaction confirmation")
        if err == nil && state != pi.TransactionStateConfirmed {
                err = errors.New("bad transaction state")
        }
        return
}

func getConn(host host.Host, protocol string,  dsn string) (c *client.Conn, err error) {
        cfg := client.NewConfig()
        cfg.DomainID = dsn

	//wyong, 20201007
	cfg.Host = host

	//wyong, 20201008
	cfg.Protocol = protocol 

        //if s.mirrorServerAddr != "" {
        //        cfg.Mirror = s.mirrorServerAddr
        //}

	//wyong, 20200819 
        //var cfg *client.Config
        //if cfg, err = client.ParseDSN(dsn); err != nil {
        //        return
        //}

	//wyong, 20201021 
        //if atomic.LoadUint32(&driverInitialized) == 0 {
        //        err = defaultInit(host)
        //        if err != nil && err != ErrAlreadyInitialized {
        //                return
        //        }
        //}

        return client.NewConn(cfg)
}


func domainForUrl(urlstring string) (domain string, err error) {
	u, err := url.Parse(urlstring)
	if err != nil {
		return "", err 
	}

	domain = u.Scheme + "://" + u.Host 
	return 
}


// readMasterKey reads the password of private key from terminal
func readMasterKey(skip bool) string {
	if skip {
		return ""
	}
	fmt.Println("Enter master key(press Enter for default: \"\"): ")
	bytePwd, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		//ConsoleLog.Errorf("read master key failed: %v", err)
		//SetExitStatus(1)
		//Exit()
		os.Exit(1)
	}
	return string(bytePwd)
}


func askDeleteFile(file string) {
        if fileinfo, err := os.Stat(file); err == nil {
                if fileinfo.IsDir() {
                        return
                }
                reader := bufio.NewReader(os.Stdin)
                fmt.Printf("\"%s\" already exists. \nDo you want to delete it? (y or n, press Enter for default n):\n",
                        file)
                t, err := reader.ReadString('\n')
                t = strings.Trim(t, "\n")
                if err != nil {
                        //ConsoleLog.WithError(err).Error("unexpected error")
                        //SetExitStatus(1)
                        //Exit()
			os.Exit(1)
			return 
                }
                if strings.EqualFold(t, "y") || strings.EqualFold(t, "yes") {
                        err = os.Remove(file)
                        if err != nil {
                                //ConsoleLog.WithError(err).Error("unexpected error")
                                //SetExitStatus(1)
                                //Exit()
				os.Exit(1)
				return 
                        }
                } else {
                        //Exit()
			os.Exit(0)
			return 
                }
        }
}


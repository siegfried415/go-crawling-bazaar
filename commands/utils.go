
package commands

import (
	"bufio" 
	"context" 
	"errors" 
        "fmt"
        "io"
	"os" 
	"strings" 
	"syscall" 
	"time" 
	url "net/url" 	

	"golang.org/x/crypto/ssh/terminal"

        client "github.com/siegfried415/go-crawling-bazaar/client"
        "github.com/siegfried415/go-crawling-bazaar/crypto/hash"
        pi "github.com/siegfried415/go-crawling-bazaar/presbyterian/interfaces"
        log "github.com/siegfried415/go-crawling-bazaar/utils/log"
        net "github.com/siegfried415/go-crawling-bazaar/net"

)

var (
        waitTxConfirmationMaxDuration time.Duration = 1 * time.Minute 
)


// PrintString prints a given Stringer to the writer.
func PrintString(w io.Writer, s fmt.Stringer) error {
        _, err := fmt.Fprintln(w, s.String())
        return err
}

func wait(host net.RoutedHost, txHash hash.Hash) (err error) {
        var ctx, cancel = context.WithTimeout(context.Background(), waitTxConfirmationMaxDuration)
        defer cancel()
        var state pi.TransactionState
        state, err = client.WaitTxConfirmation(ctx, host, txHash)
        log.WithFields(log.Fields{
                "tx_hash":  txHash,
                "tx_state": state,
        }).WithError(err).Info("wait transaction confirmation")

        if err == nil && state != pi.TransactionStateConfirmed {
                err = errors.New("bad transaction state")
        }
        return
}

func getConn(host net.RoutedHost, protocol string,  dsn string) (c *client.Conn, err error) {
        cfg := client.NewConfig()
        cfg.DomainID = dsn
	cfg.Host = host
	cfg.Protocol = protocol 

        return client.NewConn(cfg)
}

// IsURL returns true if the string represents a valid URL that the
// urlstore can handle.  More specifically it returns true if a string
// begins with 'http://' or 'https://'.
func IsURL(str string) bool {
	return (len(str) > 7 && str[0] == 'h' && str[1] == 't' && str[2] == 't' && str[3] == 'p') &&
			((len(str) > 8 && str[4] == 's' && str[5] == ':' && str[6] == '/' && str[7] == '/') ||
					(str[4] == ':' && str[5] == '/' && str[6] == '/'))
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
		log.Errorf("read master key failed: %v", err)
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
                log.Debugf("\"%s\" already exists. \nDo you want to delete it? (y or n, press Enter for default n):\n",
                        file)
                t, err := reader.ReadString('\n')
                t = strings.Trim(t, "\n")
                if err != nil {
                        log.WithError(err).Error("unexpected error")
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


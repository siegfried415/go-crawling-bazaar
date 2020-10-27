package conf

import (
	"fmt"
	//"io"
	"io/ioutil"
	"os"
	"path/filepath"
	//"strconv"
	//"strings"
	//"sync"
	//"time"

	//logging "github.com/ipfs/go-log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"

	//"github.com/siegfried415/gdf-rebuild/conf"
)

const (
	// apiFile is the filename containing the filecoin node's api address.
	apiFile                = "api"
)

//var log = logging.Logger("repo")


func RemoveAPIFile(repoPath string) error {
	if err := os.Remove(filepath.Join(repoPath, apiFile)); err != nil && !os.IsNotExist(err) {
		return err 
	}
	return nil 
}


// SetAPIAddr writes the address to the API file. SetAPIAddr expects parameter
// `port` to be of the form `:<port>`.
func SetAPIAddr(repoPath string, maddr string) error {
	f, err := os.Create(filepath.Join(repoPath, apiFile))
	if err != nil {
		return errors.Wrap(err, "could not create API file")
	}

	defer f.Close() // nolint: errcheck

	_, err = f.WriteString(maddr)
	if err != nil {
		// If we encounter an error writing to the API file,
		// delete the API file. The error encountered while
		// deleting the API file will be returned (if one
		// exists) instead of the write-error.
		if err := RemoveAPIFile(repoPath); err != nil {
			return errors.Wrap(err, "failed to remove API file")
		}

		return errors.Wrap(err, "failed to write to API file")
	}

	return nil
}


// APIAddrFromRepoPath returns the api addr from the filecoin repo
func APIAddrFromRepoPath(repoPath string) (string, error) {
	repoPath, err := homedir.Expand(repoPath)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("can't resolve local repo path %s", repoPath))
	}
	return apiAddrFromFile(filepath.Join(repoPath, apiFile))
}

// APIAddrFromFile reads the address from the API file at the given path.
// A relevant comment from a similar function at go-ipfs/repo/fsrepo/fsrepo.go:
// This is a concurrent operation, meaning that any process may read this file.
// Modifying this file, therefore, should use "mv" to replace the whole file
// and avoid interleaved read/writes
func apiAddrFromFile(apiFilePath string) (string, error) {
	contents, err := ioutil.ReadFile(apiFilePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read API file")
	}

	return string(contents), nil
}

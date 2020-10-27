package conf 

import (
	"fmt" 
	"path/filepath"

	badgerds "github.com/ipfs/go-ds-badger"
)

func badgerOptions() *badgerds.Options {
	result := &badgerds.DefaultOptions
	result.Truncate = true
	return result
}

func NewDatastore(repoPath string ) (*badgerds.Datastore, error) {
	switch GConf.Datastore.Type {
	case "badgerds":
		ds, err := badgerds.NewDatastore(filepath.Join(repoPath, GConf.Datastore.Path), badgerOptions())
		if err != nil {
			return nil, err
		}
		return ds, nil 

	default:
		return nil, fmt.Errorf("unknown datastore type in config: %s", GConf.Datastore.Type)
	}

	return nil, nil 
}


package frontera 

import (
	"context"
	//"errors"

	//notifications "github.com/siegfried415/go-crawling-market/protocol/biddingsys/notifications"

	//cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	//blockstore "gx/ipfs/QmRu7tiRnFk9mMPpVECQTBQJqXtmG132jJxA1w9A7TtpBz/go-ipfs-blockstore"
	//blocks "gx/ipfs/QmWoXtvgC8inqFkAATB7cp2Dax7XBi9VDvSg9RCCZufmRk/go-block-format"
)

type putBiddingsFunc func(context.Context, []string) (/* <-chan cid.Cid, */  error)

func putBidding(p context.Context, url string, cb putBiddingsFunc ) (error) {
	/*
	if !url.Defined() {
		log.Error("undefined cid in GetBlock")
		return nil, blockstore.ErrNotFound
	}
	*/

	// Any async work initiated by this function must end when this function
	// returns. To ensure this, derive a new context. Note that it is okay to
	// listen on parent in this scope, but NOT okay to pass |parent| to
	// functions called by this one. Otherwise those functions won't return
	// when this context's cancel func is executed. This is difficult to
	// enforce. May this comment keep you safe.
	ctx, cancel := context.WithCancel(p)
	defer cancel()

	/* promise, */  err := cb(ctx, []string{url})
	if err != nil {
		return err
	}

	/*
	select {
	case url, ok := <-promise:
		if !ok {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return errors.New("promise channel was closed")
			}
		}
		log.Debug("createBidding for %s", url)
		return nil
	case <-p.Done():
		return p.Err()
	}
	*/

	return nil 
}

type wantFunc func(context.Context, []string )

func putBiddingsImpl(ctx context.Context, urls []string, /* notif notifications.PubSub, */  want wantFunc, cwants func([]string)) (/* <-chan blocks.Block, */  error) {
	if len(urls) == 0 {
		//out := make(chan blocks.Block)
		//close(out)
		//return out, nil
		return nil
	}

	/* wyong, 20190118
	remaining := cid.NewSet()
	promise := notif.Subscribe(ctx, keys...)
	for _, u := range urls {
		//log.Event(ctx, "Biddingsys.PutBiddingRequest.Start", u)
		remaining.Add(u)
	}
	*/

	want(ctx, urls)

	//out := make(chan blocks.Block)
	//go handleIncoming(ctx, remaining, promise, out, cwants)
	//return out, nil

	return nil 
}

/*
func handleIncoming(ctx context.Context, remaining *cid.Set, in <-chan blocks.Block, out chan blocks.Block, cfun func([]cid.Cid)) {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		close(out)
		// can't just defer this call on its own, arguments are resolved *when* the defer is created
		cfun(remaining.Keys())
	}()
	for {
		select {
		case blk, ok := <-in:
			if !ok {
				return
			}

			remaining.Remove(blk.Cid())
			select {
			case out <- blk:
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
*/


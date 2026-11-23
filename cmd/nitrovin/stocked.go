package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strconv"

	"github.com/joshuasprow/nitrovin/pkg/contract"
)

func init() {
	register(command{
		name:     "stocked",
		synopsis: "[flags]",
		summary:  "List the vehicles the loaded yards hold most of, each model once in its best year, each with a VIN of the yard's to decode, translate or fit from.",
		setup:    stockedSetup,
	})
}

// stockedSetup is `nitrovin stocked`: one stocked request, of the flags,
// answered as one object.
func stockedSetup(fs *flag.FlagSet) func(*env, []string) error {
	var req contract.StockedRequest
	fs.Func("location", "keep the yard rows at `location`; repeatable", func(s string) error {
		if s == "" {
			return errors.New("must not be empty")
		}
		req.Location = append(req.Location, s)
		return nil
	})
	fs.Func("limit", fmt.Sprintf("at most `N` vehicles, 1 to %d (default %d)", contract.MaxStockedLimit, contract.DefaultStockedLimit), func(s string) error {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > contract.MaxStockedLimit {
			return fmt.Errorf("must be an integer from 1 to %d", contract.MaxStockedLimit)
		}
		req.Limit = n
		return nil
	})
	return func(e *env, args []string) error {
		if len(args) > 0 {
			return usageError("stocked takes no argument: it lists what the loaded yards hold")
		}
		t, err := e.translator()
		if err != nil {
			return err
		}
		s, err := t.Stocked(context.Background(), req)
		if err != nil {
			if p := contract.AsProblem(err); p.Type == contract.InvalidRequest {
				return refusal{p}
			}
			return err
		}
		return writeObject(e, s)
	}
}

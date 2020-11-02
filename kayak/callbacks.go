/*
 * Copyright 2019 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kayak

import (
	"context"

	"github.com/pkg/errors"

	"github.com/siegfried415/gdf-rebuild/utils/trace"

	//wyong, 20200612
        "github.com/siegfried415/gdf-rebuild/utils/log"

)


func (r *Runtime) doCheck(ctx context.Context, req interface{}) (err error) {
	defer trace.StartRegion(ctx, "checkCallback").End()
	if err = r.sh.Check(req); err != nil {
		err = errors.Wrap(err, "verify log")
	}

	return
}

func (r *Runtime) doEncodePayload(ctx context.Context, req interface{}) (enc []byte, err error) {
	defer trace.StartRegion(ctx, "encodePayloadCallback").End()
	if enc, err = r.sh.EncodePayload(req); err != nil {
		err = errors.Wrap(err, "encode kayak payload failed")
	}
	return
}

func (r *Runtime) doDecodePayload(ctx context.Context, data []byte) (req interface{}, err error) {
	defer trace.StartRegion(ctx, "decodePayloadCallback").End()
	if req, err = r.sh.DecodePayload(data); err != nil {
		err = errors.Wrap(err, "decode kayak payload failed")
	}
	return
}

func (r *Runtime) doCommit(ctx context.Context, req interface{}, isLeader bool) (result interface{}, err error) {
	defer trace.StartRegion(ctx, "commitCallback").End()

	//wyong, 20200612
	log.WithFields(log.Fields{
		"req":       req,
		"isLeader":        isLeader,
		}).Debug("kayak/runtime, doCommit() called")

	return r.sh.Commit(req, isLeader)
}
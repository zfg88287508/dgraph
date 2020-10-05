/*
 * Copyright 2020 Dgraph Labs, Inc. and Contributors
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

package admin

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/pkg/errors"

	"github.com/dgraph-io/dgraph/graphql/resolve"
	"github.com/dgraph-io/dgraph/graphql/schema"
	"github.com/dgraph-io/dgraph/protos/pb"
	"github.com/dgraph-io/dgraph/worker"
)

type moveTabletInput struct {
	Tablet string
	// TODO: once we have type for uint32 available in admin schema,
	// update the type of this field from Int64 with the new type name in admin schema.
	GroupId uint32
}

func resolveMoveTablet(ctx context.Context, m schema.Mutation) (*resolve.Resolved, bool) {
	input, err := getMoveTabletInput(m)
	if err != nil {
		return resolve.EmptyResult(m, err), false
	}

	// move tablet won't return a nil status, unless there was some gRPC connection issue
	status, err := worker.MoveTabletOverNetwork(ctx, &pb.MoveTabletRequest{Tablet: input.Tablet,
		DstGroup: input.GroupId})

	code := "Success"
	resolverSucceeded := true
	if err != nil {
		code = "Failure"
		resolverSucceeded = false
	}

	return &resolve.Resolved{
		Data:  map[string]interface{}{m.Name(): response(code, status.GetMsg())},
		Field: m,
		Err:   schema.GQLWrapLocationf(err, m.Location(), "resolving %s failed", m.Name()),
	}, resolverSucceeded
}

func getMoveTabletInput(m schema.Mutation) (*moveTabletInput, error) {
	inputArg, ok := m.ArgValue(schema.InputArgName).(map[string]interface{})
	if !ok {
		return nil, inputArgError(errors.Errorf("can't convert input to map"))
	}

	inputRef := &moveTabletInput{}
	inputRef.Tablet, ok = inputArg["tablet"].(string)
	if !ok {
		return nil, inputArgError(errors.Errorf("can't convert input.tablet to string"))
	}

	gId, err := getInt64FieldAsUint32(inputArg["groupId"])
	if err != nil {
		return nil, inputArgError(schema.GQLWrapf(err, "can't convert input.groupId to uint32"))
	}
	inputRef.GroupId = gId

	return inputRef, nil
}

func inputArgError(err error) error {
	return schema.GQLWrapf(err, "couldn't get input argument")
}

func getInt64FieldAsUint32(val interface{}) (uint32, error) {
	gId := uint64(0)
	var err error

	switch v := val.(type) {
	case string:
		gId, err = strconv.ParseUint(v, 10, 32)
	case json.Number:
		gId, err = strconv.ParseUint(v.String(), 10, 32)
	default:
		err = errors.Errorf("got unexpected value type")
	}

	return uint32(gId), err
}

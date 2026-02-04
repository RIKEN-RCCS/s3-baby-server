// deleting.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Deletion.  This is the common part of {DeleteObject,
// DeleteObjects}.

package server

import (
	"context"
	"os"
)

func (bbs *Bb_server) delete_object(ctx context.Context, object string, conditionals copy_conditionals) *Aws_s3_error {
	var location = "/" + object
	var _, rid = get_action_name(ctx)
	// var rid uint64 = get_request_id(ctx)

	var object_etag string
	var object_entity string
	{
		var stat1, entity1, err12 = bbs.fetch_object_status(object, false)
		if err12 != nil {
			// IGNORE-ERRORS.
		}
		if stat1 == nil {
			object_etag = ""
			object_entity = ""
		} else {
			var etag1, err31 = bbs.fetch_object_etag(object)
			if err31 != nil {
				// IGNORE-ERRORS.
			}
			object_etag = etag1
			object_entity = entity1
		}
	}

	// SERIALIZE-ACCESSES.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	{
		var _, err3 = bbs.check_object_exists(object)
		if err3 != nil {
			return err3
		}

		var _, entity2, err12 = bbs.fetch_object_status(object, true)
		if err12 != nil {
			// IGNORE-ERRORS.
		}
		if object_entity != entity2 {
			// The target object changed before/after serialization.
			bbs.logger.Error("RACE: Target object gone while serialized",
				"object", object)
			var errz = &Aws_s3_error{Code: InternalError,
				Message:  "Target object gone.",
				Resource: location}
			return errz
		}

		var err5 = bbs.check_conditionals(object, object_etag, "delete",
			conditionals)
		if err5 != nil {
			return err5
		}

		var err1 = bbs.store_metainfo(object, nil)
		if err1 != nil {
			// IGNORE-ERRORS.
		}
		var path = bbs.make_path_of_object(object, "")
		var err2 = os.Remove(path)
		if err2 != nil {
			bbs.logger.Warn("os.Remove() failed on an object",
				"path", path, "error", err2)
			return map_os_error(location, err2, nil)
		}
	}

	return nil
}
